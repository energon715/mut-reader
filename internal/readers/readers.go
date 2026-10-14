// Package readers converts untrusted EPUB/FB2 into a plain, allowlisted document
// model. It never returns source HTML, scripts, styles, or external URLs.
package readers

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const MaxBookBytes = 64 << 20
const MaxEntryBytes = 8 << 20
const MaxExpandedBytes = 128 << 20
const MaxBlocks = 100000

type Block struct {
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	Image string `json:"image,omitempty"`
}
type Chapter struct {
	Title  string  `json:"title"`
	Blocks []Block `json:"blocks"`
}
type Document struct {
	Format   string    `json:"format"`
	Chapters []Chapter `json:"chapters"`
	PDF      string    `json:"pdf,omitempty"`
	Cover    string    `json:"cover,omitempty"`
}

func Parse(format string, data []byte) (Document, error) {
	if len(data) > MaxBookBytes {
		return Document{}, errors.New("reader limit: 64 MiB per book")
	}
	switch strings.ToUpper(format) {
	case "EPUB":
		return parseEPUB(data)
	case "FB2":
		return parseFB2(data)
	case "PDF":
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return Document{}, errors.New("invalid PDF")
		}
		return Document{Format: "PDF", Chapters: []Chapter{}, PDF: base64.StdEncoding.EncodeToString(data)}, nil
	default:
		return Document{}, errors.New("no built-in reader")
	}
}
func imageURI(data []byte) string {
	if len(data) > 5<<20 {
		return ""
	}
	mime := http.DetectContentType(data)
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/gif" {
		return ""
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 16000000 {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}
func resolve(base, href string) (string, error) {
	u, err := url.Parse(href)
	if err != nil || u.IsAbs() || u.Host != "" {
		return "", errors.New("external book resource blocked")
	}
	p := path.Clean(path.Join(path.Dir(base), u.Path))
	if !safePath(p) {
		return "", errors.New("invalid book resource path")
	}
	return p, nil
}
func safePath(p string) bool {
	return p != "." && !strings.HasPrefix(p, "/") && !strings.Contains(p, "\\") && p != ".." && !strings.HasPrefix(p, "../") && path.Clean(p) == p
}
func decoder(data []byte) *xml.Decoder {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	return d
}
func parseEPUB(data []byte) (Document, error) {
	doc := Document{Format: "EPUB", Chapters: []Chapter{}}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return doc, err
	}
	if len(z.File) > 4096 {
		return doc, errors.New("too many EPUB entries")
	}
	files := map[string]*zip.File{}
	var expanded uint64
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if !safePath(f.Name) || f.Mode()&0x08000000 != 0 {
			return doc, errors.New("invalid EPUB entry")
		}
		if _, ok := files[f.Name]; ok {
			return doc, errors.New("duplicate EPUB entry")
		}
		expanded += f.UncompressedSize64
		if f.UncompressedSize64 > MaxEntryBytes || expanded > MaxExpandedBytes {
			return doc, errors.New("EPUB expansion limit exceeded")
		}
		files[f.Name] = f
	}
	budget := &contentBudget{}
	readBytes := 0
	read := func(name string) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, errors.New("missing EPUB resource")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		b, err := io.ReadAll(io.LimitReader(r, MaxEntryBytes+1))
		if len(b) > MaxEntryBytes {
			return nil, errors.New("EPUB entry too large")
		}
		readBytes += len(b)
		if readBytes > MaxExpandedBytes {
			return nil, errors.New("repeated EPUB resource read limit exceeded")
		}
		return b, err
	}
	container, err := read("META-INF/container.xml")
	if err != nil {
		return doc, err
	}
	var c struct {
		Root struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := DecodeXML(container, &c); err != nil {
		return doc, err
	}
	if !safePath(c.Root.Path) {
		return doc, errors.New("invalid OPF path")
	}
	opf, err := read(c.Root.Path)
	if err != nil {
		return doc, err
	}
	type item struct {
		ID         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		Type       string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
	}
	var pkg struct {
		Items []item `xml:"manifest>item"`
		Spine []struct {
			ID string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
		Meta []struct {
			Name    string `xml:"name,attr"`
			Content string `xml:"content,attr"`
		} `xml:"metadata>meta"`
	}
	if err := DecodeXML(opf, &pkg); err != nil {
		return doc, err
	}
	if len(pkg.Spine) > 4096 {
		return doc, errors.New("too many EPUB spine entries")
	}
	manifest := map[string]item{}
	coverID := ""
	for _, m := range pkg.Meta {
		if m.Name == "cover" {
			coverID = m.Content
		}
	}
	for _, i := range pkg.Items {
		manifest[i.ID] = i
		if i.ID == coverID || strings.Contains(i.Properties, "cover-image") {
			if name, err := resolve(c.Root.Path, i.Href); err == nil {
				if b, err := read(name); err == nil {
					doc.Cover = imageURI(b)
				}
			}
		}
	}
	imageCache := map[string]string{}
	blocks := 0
	for _, ref := range pkg.Spine {
		item, ok := manifest[ref.ID]
		if !ok {
			return doc, errors.New("invalid EPUB spine")
		}
		name, err := resolve(c.Root.Path, item.Href)
		if err != nil {
			return doc, err
		}
		source, err := read(name)
		if err != nil {
			return doc, err
		}
		chapter, err := htmlChapter(source, name, budget, func(href string) string {
			p, err := resolve(name, href)
			if err != nil {
				return ""
			}
			if files[p] == nil {
				return ""
			}
			if uri, ok := imageCache[p]; ok {
				return uri
			}
			b, err := read(p)
			if err != nil {
				budget.err = err
				return ""
			}
			uri := imageURI(b)
			imageCache[p] = uri
			return uri
		})
		if err != nil {
			return doc, err
		}
		if len(chapter.Blocks) == 0 {
			continue
		}
		blocks += len(chapter.Blocks)
		if blocks > MaxBlocks {
			return doc, errors.New("too many book blocks")
		}
		if chapter.Title == "" {
			chapter.Title = fmt.Sprintf("Глава %d", len(doc.Chapters)+1)
		}
		doc.Chapters = append(doc.Chapters, chapter)
	}
	if !budget.take(len(doc.Cover)) {
		return doc, budget.err
	}
	if len(doc.Chapters) == 0 {
		return doc, errors.New("EPUB contains no readable chapters")
	}
	return doc, nil
}
func htmlChapter(source []byte, name string, budget *contentBudget, img func(string) string) (Chapter, error) {
	chapter := Chapter{Blocks: []Block{}}
	r, err := charset.NewReader(bytes.NewReader(source), "")
	if err != nil {
		return chapter, err
	}
	tokens := html.NewTokenizer(r)
	tokens.SetMaxBuf(MaxEntryBytes)
	var text strings.Builder
	kind := "p"
	skip := 0
	depth := 0
	count := 0
	flush := func() {
		v := strings.Join(strings.Fields(text.String()), " ")
		text.Reset()
		if v != "" {
			if !budget.block(len(v)) {
				return
			}
			chapter.Blocks = append(chapter.Blocks, Block{Kind: kind, Text: v})
			if chapter.Title == "" && (kind == "h1" || kind == "h2") {
				if !budget.take(len(v)) {
					return
				}
				chapter.Title = v
			}
		}
		kind = "p"
	}
	forbidden := func(tag string) bool {
		switch tag {
		case "script", "style", "iframe", "object", "embed", "form", "head", "svg", "math":
			return true
		}
		return false
	}
	block := func(tag string) bool {
		switch tag {
		case "p", "div", "li", "section", "h1", "h2", "h3", "blockquote", "pre":
			return true
		}
		return false
	}
	for {
		if budget.err != nil {
			return chapter, budget.err
		}
		if !budget.token() {
			return chapter, budget.err
		}
		typ := tokens.Next()
		count++
		if count > 500000 || len(chapter.Blocks) > MaxBlocks {
			return chapter, errors.New("HTML complexity limit exceeded")
		}
		switch typ {
		case html.ErrorToken:
			if tokens.Err() != io.EOF {
				return chapter, tokens.Err()
			}
			flush()
			return chapter, budget.err
		case html.StartTagToken, html.SelfClosingTagToken:
			t := tokens.Token()
			if typ == html.StartTagToken && !voidTag(t.Data) {
				depth++
				if depth > 256 {
					return chapter, errors.New("HTML depth limit exceeded")
				}
			}
			if skip > 0 {
				if typ == html.StartTagToken && !voidTag(t.Data) {
					skip++
				}
				continue
			}
			if forbidden(t.Data) {
				if typ == html.StartTagToken && !voidTag(t.Data) {
					skip = 1
				}
				continue
			}
			if block(t.Data) {
				flush()
				if strings.HasPrefix(t.Data, "h") {
					kind = "h2"
				}
				if t.Data == "blockquote" {
					kind = "quote"
				}
				if t.Data == "pre" {
					kind = "pre"
				}
			}
			if t.Data == "br" {
				text.WriteByte(' ')
			}
			if t.Data == "img" {
				flush()
				for _, a := range t.Attr {
					if a.Key == "src" {
						if uri := img(a.Val); uri != "" {
							if !budget.block(len(uri)) {
								return chapter, budget.err
							}
							chapter.Blocks = append(chapter.Blocks, Block{Kind: "image", Image: uri})
						}
					}
				}
			}
		case html.EndTagToken:
			t := tokens.Token()
			if depth > 0 {
				depth--
			}
			if skip > 0 {
				skip--
				continue
			}
			if block(t.Data) {
				flush()
			}
		case html.TextToken:
			if skip == 0 {
				text.Write(tokens.Text())
				if text.Len() > MaxEntryBytes {
					return chapter, errors.New("text block too large")
				}
			}
		}
	}
}
func parseFB2(data []byte) (Document, error) {
	doc := Document{Format: "FB2", Chapters: []Chapter{}}
	d := decoder(data)
	chapter := Chapter{Title: "Начало", Blocks: []Block{}}
	images := map[string]string{}
	body := 0
	depth := 0
	budget := &contentBudget{}
	inTitle := false
	coverID := ""
	inCover := false
	flushChapter := func() {
		if len(chapter.Blocks) > 0 {
			if !budget.take(len(chapter.Title)) {
				return
			}
			doc.Chapters = append(doc.Chapters, chapter)
			chapter = Chapter{Title: fmt.Sprintf("Глава %d", len(doc.Chapters)+1), Blocks: []Block{}}
		}
	}
	for {
		if budget.err != nil {
			return doc, budget.err
		}
		if !budget.token() {
			return doc, budget.err
		}
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, err
		}
		switch t := token.(type) {
		case xml.Directive:
			return doc, errors.New("XML directives are not supported")
		case xml.StartElement:
			depth++
			if depth > 128 {
				return doc, errors.New("XML depth limit exceeded")
			}
			switch t.Name.Local {
			case "body":
				body++
			case "coverpage":
				inCover = true
			case "image":
				for _, a := range t.Attr {
					if a.Name.Local == "href" && strings.HasPrefix(a.Value, "#") {
						id := strings.TrimPrefix(a.Value, "#")
						if inCover {
							coverID = id
						}
						if body > 0 {
							if !budget.block(len(id)) {
								return doc, budget.err
							}
							chapter.Blocks = append(chapter.Blocks, Block{Kind: "image", Image: id})
						}
					}
				}
			case "binary":
				var id string
				for _, a := range t.Attr {
					if a.Name.Local == "id" {
						id = a.Value
					}
				}
				value, err := xmlText(d, t, depth, budget)
				if err != nil {
					return doc, err
				}
				depth--
				if len(value) > 8<<20 {
					return doc, errors.New("FB2 image too large")
				}
				b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
				if err == nil {
					images[id] = imageURI(b)
				}
				clear(b)
			case "title":
				if body > 0 {
					flushChapter()
					inTitle = true
				}
			case "p", "subtitle", "text-author", "v":
				if body > 0 {
					v, err := xmlText(d, t, depth, budget)
					if err != nil {
						return doc, err
					}
					depth--
					v = strings.TrimSpace(v)
					if v != "" {
						kind := "p"
						if inTitle {
							chapter.Title = v
							kind = "h2"
						}
						if !budget.block(len(v)) {
							return doc, budget.err
						}
						chapter.Blocks = append(chapter.Blocks, Block{Kind: kind, Text: v})
					}
				}
			}
		case xml.EndElement:
			depth--
			switch t.Name.Local {
			case "body":
				body--
			case "title":
				inTitle = false
			case "coverpage":
				inCover = false
			}
		}
	}
	flushChapter()
	if budget.err != nil {
		return doc, budget.err
	}
	if !budget.take(len(images[coverID])) {
		return doc, budget.err
	}
	for i := range doc.Chapters {
		for j := range doc.Chapters[i].Blocks {
			b := &doc.Chapters[i].Blocks[j]
			if b.Kind == "image" {
				uri := images[b.Image]
				if len(uri) > len(b.Image) && !budget.take(len(uri)-len(b.Image)) {
					return doc, budget.err
				}
				b.Image = uri
			}
		}
	}
	doc.Cover = images[coverID]
	if len(doc.Chapters) == 0 {
		return doc, errors.New("FB2 contains no readable text")
	}
	return doc, nil
}
func xmlText(d *xml.Decoder, start xml.StartElement, outerDepth int, budget *contentBudget) (string, error) {
	var out strings.Builder
	depth := 1
	for depth > 0 {
		if !budget.token() {
			return "", budget.err
		}
		token, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			if depth+outerDepth-1 > 128 {
				return "", errors.New("XML depth limit exceeded")
			}
		case xml.EndElement:
			depth--
		case xml.CharData:
			out.Write(t)
			if out.Len() > MaxEntryBytes {
				return "", errors.New("paragraph too large")
			}
		case xml.Directive:
			return "", errors.New("XML directives are not supported")
		}
	}
	return out.String(), nil
}

func voidTag(tag string) bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}
