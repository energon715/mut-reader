package importer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"mut/internal/catalog"
	"mut/internal/vault"
	"path/filepath"
	"strings"
	"uuid"
)

type fb2book struct {
	XMLName xml.Name    `xml:"FictionBook"`
	Title   string      `xml:"description>title-info>book-title"`
	Author  []fb2Author `xml:"description>title-info>author"`
}

type fb2Author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
}

type epubContainer struct {
	RootFile struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Title    string   `xml:"metadata>title"`
	Creators []string `xml:"metadata>creator"`
}

type pdfXMP struct {
	Title  string `xml:"RDF>Description>title>Alt>li"`
	Author string `xml:"RDF>Description>creator>Seq>li"`
}

var ErrDuplicateBook = errors.New("book already exists")

func ImportFile(filename string, v *vault.Vault, c *catalog.Catalog, data []byte, keepDuplicate bool) error {
	sum := sha256.Sum256(data)
	sha256 := hex.EncodeToString(sum[:])
	book, _, err := c.FindByHash(sha256)
	if err != nil {
		return fmt.Errorf("find by hash error: %v", err)
	}
	if book.ID != uuid.Nil() && !keepDuplicate {
		return ErrDuplicateBook
	}
	objectID, err := v.PutObject(data)
	if err != nil {
		return fmt.Errorf("cannot save object error: %w", err)
	}
	ext := filepath.Ext(filename)
	book.FileObjectID = objectID
	book.Sha256 = sha256
	book.Originalname = filename
	book.FileSize = len(data)
	book.ID = uuid.New()
	book.Format = strings.ToUpper(strings.TrimLeft(ext, "."))
	if book.Format == "FB2" {
		title, authorname, err := ParseFB2(data)
		if err != nil {
			log.Printf("cannot parse fb2 error: %v", err)
		}
		if book.Title == "" {
			book.Title = title
		}
		if book.Author == "" {
			book.Author = authorname
		}
	} else if book.Format == "EPUB" {
		title, authorname, err := ParseEPUB(data)
		if err != nil {
			log.Printf("cannot parse epub error: %v", err)
		}
		if book.Title == "" {
			book.Title = title
		}
		if book.Author == "" {
			book.Author = authorname
		}
	} else if book.Format == "PDF" {
		title, authorname, err := ParsePDF(data)
		if err != nil {
			log.Printf("cannot parse pdf error: %v", err)
		}
		if book.Title == "" {
			book.Title = title
		}
		if book.Author == "" {
			book.Author = authorname
		}
	} else {
		book.Title = filename
		book.Author = "Unknown author"
	}
	err = c.AddBook(book)
	if err != nil {
		return fmt.Errorf("cannot add book error: %w", err)
	}
	_, err = c.Save()
	if err != nil {
		return fmt.Errorf("cannot save catalog error: %w", err)
	}

	return nil

}

func ParseFB2(data []byte) (title string, author string, err error) {
	var fb2 fb2book
	err = xml.Unmarshal(data, &fb2)
	if err != nil {
		return "", "", fmt.Errorf("cannot unmarshal fb2 error: %w", err)
	}

	var authors []string
	var authorname string

	for _, author := range fb2.Author {
		authorname = ""
		if author.LastName != "" {
			authorname += author.LastName
		}
		if author.FirstName != "" {
			authorname += " " + author.FirstName
		}
		if author.MiddleName != "" {
			authorname += " " + author.MiddleName
		}
		authors = append(authors, authorname)
	}

	title = strings.TrimSpace(fb2.Title)
	if title == "" {
		title = "unknown title"
	}
	if len(authors) == 0 {
		author = "unknown author"
	} else {
		author = strings.TrimSpace(strings.Join(authors, ", "))
	}
	return title, author, nil
}

func ParseEPUB(data []byte) (title string, author string, err error) {
	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", "", fmt.Errorf("cannot read zip/epub error: %v", err)
	}

	for _, file := range zipReader.File {
		if strings.EqualFold(file.Name, "META-INF/container.xml") {
			rc, err := file.Open()
			if err != nil {
				return "", "", fmt.Errorf("cannot open container.xml error: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(io.LimitReader(rc, 2*1024*1024))
			if err != nil {
				return "", "", fmt.Errorf("cannot read first 2mb of epub error: %v", err)
			}

			var container epubContainer
			err = xml.Unmarshal(content, &container)
			if err != nil {
				return "", "", fmt.Errorf("cannot unmarshal container.xml error: %v", err)
			}

			for _, file := range zipReader.File {
				if file.Name == container.RootFile.FullPath {
					rc, err := file.Open()
					if err != nil {
						return "", "", fmt.Errorf("cannot open opf error: %v", err)
					}
					defer rc.Close()

					content, err := io.ReadAll(io.LimitReader(rc, 2*1024*1024))
					if err != nil {
						return "", "", fmt.Errorf("cannot read first 2mb of epub error: %v", err)
					}

					var opf opfPackage
					err = xml.Unmarshal(content, &opf)
					if err != nil {
						return "", "", fmt.Errorf("cannot unmarshal opf error: %v", err)
					}

					title = strings.TrimSpace(opf.Title)
					if title == "" {
						title = "unknown title"
					}

					author = strings.TrimSpace(strings.Join(opf.Creators, ", "))
					if author == "" {
						author = "unknown author"
					}

					return title, author, nil
				}
			}
		}
	}
	return "", "", fmt.Errorf("container.xml or opf not found")
}

func ParsePDF(data []byte) (title string, author string, err error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return "", "", errors.New("not pdf file")
	}

	startIDX := bytes.Index(data, []byte("<x:xmpmeta"))
	endIDX := bytes.Index(data, []byte("</x:xmpmeta>"))

	if startIDX != -1 && endIDX != -1 && endIDX > startIDX {
		metadata := data[startIDX : endIDX+len("</x:xmpmeta>")]

		var pdfxmp pdfXMP
		if err := xml.Unmarshal(metadata, &pdfxmp); err == nil {
			title = strings.TrimSpace(pdfxmp.Title)
			author = strings.TrimSpace(pdfxmp.Author)
		}
	}

	if title == "" {
		if t, err := ExtractPDFInfoField(data, "/Title"); err == nil {
			title = t
		}
	}
	if author == "" {
		if a, err := ExtractPDFInfoField(data, "/Author"); err == nil {
			author = a
		}
	}

	return title, author, nil
}

func ExtractPDFInfoField(data []byte, field string) (string, error) {
	idx := bytes.Index(data, []byte(field))
	if idx == -1 {
		return "", fmt.Errorf("field not found: %s", field)
	}
	p := idx + len(field)

	for p < len(data) && (data[p] == ' ' || data[p] == '\t' || data[p] == '\n' || data[p] == '\r') {
		p++
	}
	if p >= len(data) {
		return "", fmt.Errorf("unexpected end of data after %s", field)
	}

	if data[p] == '(' {
		start := p + 1
		depth := 1
		end := start
		for end < len(data) && depth > 0 {
			if data[end] == '\\' {
				end += 2
				continue
			}
			if data[end] == '(' {
				depth++
			} else if data[end] == ')' {
				depth--
				if depth == 0 {
					break
				}
			}
			end++
		}
		if depth > 0 {
			return "", fmt.Errorf("unclosed parenthesis")
		}
		raw := data[start:end]
		return decodePDFText(raw), nil
	}

	if data[p] == '<' {
		start := p + 1
		end := bytes.IndexByte(data[start:], '>')
		if end == -1 {
			return "", fmt.Errorf("unclosed angle bracket")
		}
		hexStr := strings.TrimSpace(string(data[start : start+end]))
		raw, err := hex.DecodeString(hexStr)
		if err != nil {
			return "", fmt.Errorf("invalid hex string: %w", err)
		}
		return decodePDFText(raw), nil
	}

	return "", fmt.Errorf("unsupported value format for %s", field)
}

func decodePDFText(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		runes := make([]rune, 0, (len(raw)-2)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			char := rune(uint16(raw[i])<<8 | uint16(raw[i+1]))
			runes = append(runes, char)
		}
		return strings.TrimSpace(string(runes))
	}

	var sb strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case '(', ')', '\\':
				sb.WriteByte(raw[i])
			default:
				sb.WriteByte(raw[i])
			}
		} else {
			sb.WriteByte(raw[i])
		}
	}
	return strings.TrimSpace(sb.String())
}

func ExportOriginal(v *vault.Vault, book catalog.Book) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("vault is nil")
	}
	data, err := v.GetObject(book.FileObjectID)
	if err != nil {
		return nil, fmt.Errorf("cannot get object: %v", err)
	}
	shasum := sha256.Sum256(data)

	sha256 := hex.EncodeToString(shasum[:])
	if sha256 != book.Sha256 {
		return nil, fmt.Errorf("sha256 not match")
	}

	return data, nil

}
