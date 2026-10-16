package readers

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func fixtureEPUB(t *testing.T, chapter string) []byte {
	t.Helper()
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	files := map[string]string{"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/package.opf"/></rootfiles></container>`, "OEBPS/package.opf": `<package><manifest><item id="c1" href="chapter.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`, "OEBPS/chapter.xhtml": chapter}
	for name, value := range files {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(value))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func TestEPUBSafeModel(t *testing.T) {
	data := fixtureEPUB(t, `<html><head><meta name="x"/><script>head-secret</script></head><body><h1>Chapter one</h1><p>Hello <em>reader</em>.</p><script>book-secret</script><iframe src="https://example.test"></iframe><img src="https://example.test/a.png"/><p>After image.</p></body></html>`)
	doc, err := Parse("EPUB", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Chapters) != 1 || doc.Chapters[0].Title != "Chapter one" {
		t.Fatalf("invalid chapters: %+v", doc)
	}
	for _, block := range doc.Chapters[0].Blocks {
		if strings.Contains(block.Text, "secret") || strings.Contains(block.Image, "example.test") {
			t.Fatal("active or external content escaped sanitization")
		}
	}
	if !strings.Contains(doc.Chapters[0].Blocks[len(doc.Chapters[0].Blocks)-1].Text, "After image") {
		t.Fatal("void HTML element swallowed following text")
	}
}
func TestFB2TextAndChapters(t *testing.T) {
	doc, err := Parse("FB2", []byte(`<FictionBook><body><section><title><p>First chapter</p></title><p>Hello <strong>reader</strong>.</p></section><section><title><p>Second chapter</p></title><p>Goodbye.</p></section></body></FictionBook>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Chapters) != 2 || doc.Chapters[0].Title != "First chapter" || doc.Chapters[1].Title != "Second chapter" {
		t.Fatalf("unexpected chapters: %+v", doc)
	}
}
func TestRejectXMLDirectives(t *testing.T) {
	if _, err := Parse("FB2", []byte(`<!DOCTYPE x [<!ENTITY a "attack">]><FictionBook><body><p>&a;</p></body></FictionBook>`)); err == nil {
		t.Fatal("DTD accepted")
	}
}
func TestRejectEscapingEPUBPaths(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, _ := z.Create("../outside.xhtml")
	f.Write([]byte("outside"))
	z.Close()
	if _, err := Parse("EPUB", b.Bytes()); err == nil {
		t.Fatal("escaping zip path accepted")
	}
}
func FuzzFB2(f *testing.F) {
	f.Add([]byte(`<FictionBook><body><p>Safe text</p></body></FictionBook>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		Parse("FB2", data)
	})
}

func FuzzEPUB(f *testing.F) {
	f.Add([]byte("PK"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = Parse("EPUB", data)
	})
}

func TestRepeatedSpineModelLimit(t *testing.T) {
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	files := map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="book.opf"/></rootfiles></container>`,
		"book.opf":               `<package><manifest><item id="c" href="c.xhtml"/></manifest><spine>` + strings.Repeat(`<itemref idref="c"/>`, 100) + `</spine></package>`,
		"c.xhtml":                "<html><body><p>" + strings.Repeat("x", 1<<20) + "</p></body></html>",
	}
	for name, data := range files {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("EPUB", out.Bytes()); err == nil || !strings.Contains(err.Error(), "model limit") {
		t.Fatalf("repeated chapter not bounded: %v", err)
	}
}

func TestRepeatedImagesModelLimit(t *testing.T) {
	// A tiny valid raster followed by ignored trailing data keeps ZIP/input small,
	// but its Base64 representation repeated in the bridge model must be bounded.
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9ZQmcAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	pngBytes = append(pngBytes, bytes.Repeat([]byte{0}, 1<<20)...)
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for name, data := range map[string][]byte{
		"META-INF/container.xml": []byte(`<container><rootfiles><rootfile full-path="book.opf"/></rootfiles></container>`),
		"book.opf":               []byte(`<package><manifest><item id="c" href="c.xhtml"/></manifest><spine><itemref idref="c"/></spine></package>`),
		"c.xhtml":                []byte("<html><body><p>Image fixture</p>" + strings.Repeat(`<img src="image.png"/>`, 40) + "</body></html>"),
		"image.png":              pngBytes,
	} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("EPUB", out.Bytes()); err == nil || !strings.Contains(err.Error(), "model limit") {
		t.Fatalf("repeated image not bounded: %v", err)
	}
	fb2 := `<FictionBook><body><p>Fixture</p>` + strings.Repeat(`<image href="#img"/>`, 40) + `</body><binary id="img">` + base64.StdEncoding.EncodeToString(pngBytes) + `</binary></FictionBook>`
	if _, err := Parse("FB2", []byte(fb2)); err == nil || !strings.Contains(err.Error(), "model limit") {
		t.Fatalf("FB2 repeated image not bounded: %v", err)
	}
}

func TestFB2BinaryLimits(t *testing.T) {
	for _, binary := range []string{strings.Repeat("<nested>", 129) + "AA==" + strings.Repeat("</nested>", 129), `<!DOCTYPE x>AA==`} {
		if _, err := Parse("FB2", []byte(`<FictionBook><body><p>Fixture</p></body><binary id="img">`+binary+`</binary></FictionBook>`)); err == nil {
			t.Fatal("binary bypassed XML limits")
		}
	}
}
