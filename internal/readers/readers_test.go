package readers

import (
	"archive/zip"
	"bytes"
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
