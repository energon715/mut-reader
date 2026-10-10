package importer

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"testing"

	"mut/internal/catalog"
	"mut/internal/vault"
)

func createTestVaultAndCatalog(t *testing.T) (*vault.Vault, *catalog.Catalog) {
	t.Helper()
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "test_vault")
	password := []byte("strong-password-123")

	v, _, err := vault.CreateVault(vaultPath, password)
	if err != nil {
		t.Fatalf("CreateVault error: %v", err)
	}
	c, err := catalog.Open(v)
	if err != nil {
		t.Fatalf("Open catalog error: %v", err)
	}
	return v, c
}

func createMinimalEPUB(t *testing.T, title, author string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// META-INF/container.xml
	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	cw, err := zw.Create("META-INF/container.xml")
	if err != nil {
		t.Fatalf("failed to create container.xml in zip: %v", err)
	}
	_, _ = cw.Write([]byte(containerXML))

	// OEBPS/content.opf
	opfXML := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>` + title + `</dc:title>
    <dc:creator>` + author + `</dc:creator>
  </metadata>
</package>`
	ow, err := zw.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("failed to create content.opf in zip: %v", err)
	}
	_, _ = ow.Write([]byte(opfXML))

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close epub zip: %v", err)
	}
	return buf.Bytes()
}

func TestParseFB2(t *testing.T) {
	fb2Data := []byte(`<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Солярис</book-title>
      <author>
        <first-name>Станислав</first-name>
        <last-name>Лем</last-name>
      </author>
    </title-info>
  </description>
  <body><p>Sample text</p></body>
</FictionBook>`)

	title, author, err := ParseFB2(fb2Data)
	if err != nil {
		t.Fatalf("ParseFB2 error: %v", err)
	}
	if title != "Солярис" {
		t.Errorf("expected title 'Солярис', got '%s'", title)
	}
	if author != "Лем Станислав" {
		t.Errorf("expected author 'Лем Станислав', got '%s'", author)
	}
}

func TestParseEPUB(t *testing.T) {
	epubBytes := createMinimalEPUB(t, "1984", "George Orwell")

	title, author, err := ParseEPUB(epubBytes)
	if err != nil {
		t.Fatalf("ParseEPUB error: %v", err)
	}
	if title != "1984" {
		t.Errorf("expected title '1984', got '%s'", title)
	}
	if author != "George Orwell" {
		t.Errorf("expected author 'George Orwell', got '%s'", author)
	}
}

func TestParsePDF_InfoDictionary(t *testing.T) {
	pdfData := []byte("%PDF-1.4\n1 0 obj\n<< /Title (The Linux Programming Interface) /Author (Michael Kerrisk) >>\nendobj\n%%EOF")

	title, author, err := ParsePDF(pdfData)
	if err != nil {
		t.Fatalf("ParsePDF error: %v", err)
	}
	if title != "The Linux Programming Interface" {
		t.Errorf("expected title 'The Linux Programming Interface', got '%s'", title)
	}
	if author != "Michael Kerrisk" {
		t.Errorf("expected author 'Michael Kerrisk', got '%s'", author)
	}
}

func TestParsePDF_HexUnicode(t *testing.T) {
	// "Hi" в UTF-16BE: 0xFEFF, 0x0048 ('H'), 0x0069 ('i') -> FEFF00480069
	pdfData := []byte("%PDF-1.7\n<< /Title <FEFF00480069> >>\n%%EOF")

	title, _, err := ParsePDF(pdfData)
	if err != nil {
		t.Fatalf("ParsePDF error: %v", err)
	}
	if title != "Hi" {
		t.Errorf("expected title 'Hi', got '%s'", title)
	}
}

func TestParsePDF_XMP(t *testing.T) {
	pdfData := []byte(`%PDF-1.7
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/">
   <dc:title><rdf:Alt><rdf:li>Modern Operating Systems</rdf:li></rdf:Alt></dc:title>
   <dc:creator><rdf:Seq><rdf:li>Andrew S. Tanenbaum</rdf:li></rdf:Seq></dc:creator>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
%%EOF`)

	title, author, err := ParsePDF(pdfData)
	if err != nil {
		t.Fatalf("ParsePDF error: %v", err)
	}
	if title != "Modern Operating Systems" {
		t.Errorf("expected title 'Modern Operating Systems', got '%s'", title)
	}
	if author != "Andrew S. Tanenbaum" {
		t.Errorf("expected author 'Andrew S. Tanenbaum', got '%s'", author)
	}
}

func TestParsePDF_Invalid(t *testing.T) {
	_, _, err := ParsePDF([]byte("not a pdf at all"))
	if err == nil {
		t.Fatal("expected error for non-pdf content")
	}
}

func TestImportAndExportRoundtrip(t *testing.T) {
	v, c := createTestVaultAndCatalog(t)

	pdfContent := []byte("%PDF-1.4\n<< /Title (Algorithms) /Author (Sedgewick) >>\n%%EOF")

	// 1. Успешный импорт
	err := ImportFile("algorithms.pdf", v, c, pdfContent, false)
	if err != nil {
		t.Fatalf("ImportFile error: %v", err)
	}

	books, err := c.ListBooks()
	if err != nil || len(books) != 1 {
		t.Fatalf("expected 1 book, got %d (err: %v)", len(books), err)
	}

	book := books[0]
	if book.Title != "Algorithms" {
		t.Errorf("expected title 'Algorithms', got '%s'", book.Title)
	}
	if book.Author != "Sedgewick" {
		t.Errorf("expected author 'Sedgewick', got '%s'", book.Author)
	}
	if book.Format != "PDF" {
		t.Errorf("expected format 'PDF', got '%s'", book.Format)
	}

	// 2. Проверка дубликата при keepDuplicate = false
	err = ImportFile("duplicate.pdf", v, c, pdfContent, false)
	if err != ErrDuplicateBook {
		t.Fatalf("expected ErrDuplicateBook, got: %v", err)
	}

	// 3. Импорт с keepDuplicate = true разрешён
	err = ImportFile("duplicate_allowed.pdf", v, c, pdfContent, true)
	if err != nil {
		t.Fatalf("expected success with keepDuplicate=true, got: %v", err)
	}

	allBooks, _ := c.ListBooks()
	if len(allBooks) != 2 {
		t.Fatalf("expected 2 books after duplicate import, got %d", len(allBooks))
	}

	// 4. Экспорт оригинала и сверка побайтового совпадения
	exportedData, err := ExportOriginal(v, book)
	if err != nil {
		t.Fatalf("ExportOriginal error: %v", err)
	}
	if !bytes.Equal(exportedData, pdfContent) {
		t.Fatal("exported bytes do not match original imported bytes!")
	}
}

func TestImportFormatAgnostic(t *testing.T) {
	v, c := createTestVaultAndCatalog(t)

	customDoc := []byte("This is some custom game manual in unsupported format")

	// Импортируем файл неизвестного формата (например docx)
	err := ImportFile("manual.docx", v, c, customDoc, false)
	if err != nil {
		t.Fatalf("ImportFile format agnostic error: %v", err)
	}

	books, err := c.ListBooks()
	if err != nil || len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}

	book := books[0]
	if book.Format != "DOCX" {
		t.Errorf("expected format DOCX, got %s", book.Format)
	}
	if book.Title != "manual.docx" {
		t.Errorf("expected fallback title 'manual.docx', got %s", book.Title)
	}

	// Экспортируем и проверяем целостность
	exported, err := ExportOriginal(v, book)
	if err != nil {
		t.Fatalf("ExportOriginal error: %v", err)
	}
	if !bytes.Equal(exported, customDoc) {
		t.Fatal("exported custom document bytes do not match original!")
	}
}
