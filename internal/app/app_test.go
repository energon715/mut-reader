package app

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func testEPUB(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, value := range map[string]string{"META-INF/container.xml": `<container><rootfiles><rootfile full-path="book.opf"/></rootfiles></container>`, "book.opf": `<package><metadata><title>EPUB fixture</title></metadata><manifest><item id="c" href="chapter.xhtml"/></manifest><spine><itemref idref="c"/></spine></package>`, "chapter.xhtml": `<html><body><h1>Test chapter</h1><p>Author-owned fixture text.</p></body></html>`} {
		f, _ := z.Create(name)
		f.Write([]byte(value))
	}
	z.Close()
	return b.Bytes()
}
func TestApplicationRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library")
	app := NewApp()
	key, err := app.CreateVault(path, "long-fixture-password")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Lock()
	fixtures := map[string][]byte{"book.epub": testEPUB(t), "book.fb2": []byte(`<FictionBook><description><title-info><book-title>FB2 fixture</book-title></title-info></description><body><section><title><p>First chapter</p></title><p>Author-owned fixture text.</p></section></body></FictionBook>`), "book.pdf": []byte("%PDF-1.4\n<< /Title (PDF fixture) >>\n%%EOF"), "notes.txt": []byte("opaque original")}
	inputs := t.TempDir()
	for name, data := range fixtures {
		p := filepath.Join(inputs, name)
		os.WriteFile(p, data, 0600)
		if err := app.ImportBook(p); err != nil {
			t.Fatal(err)
		}
	}
	books, err := app.ListBooks()
	if err != nil || len(books) != 4 {
		t.Fatal("missing imported books")
	}
	id := books[0].ID.String()
	if err := app.CreateShelf("Fixtures"); err != nil {
		t.Fatal(err)
	}
	if err := app.SetBookShelves(id, []string{"Fixtures"}); err != nil {
		t.Fatal(err)
	}
	if err := app.SavePosition(id, 42, "v1:text:0:0"); err != nil {
		t.Fatal(err)
	}
	if err := app.SaveBookmarks(id, []string{"v1:text:0:0"}); err != nil {
		t.Fatal(err)
	}
	for _, b := range books {
		if b.Format == "TXT" {
			continue
		}
		doc, err := app.OpenBook(b.ID.String())
		if err != nil {
			t.Fatal(err)
		}
		if doc.Format != b.Format {
			t.Fatal("reader format mismatch")
		}
	}
	if err := app.ImportBook(filepath.Join(inputs, "book.fb2")); err == nil {
		t.Fatal("duplicate not detected")
	}
	if err := app.ImportEdition(filepath.Join(inputs, "book.fb2"), true); err != nil {
		t.Fatal(err)
	}
	report, err := app.CreateBackup(t.TempDir())
	if err != nil || !report.Verified {
		t.Fatalf("backup failed: %v", err)
	}
	restore := filepath.Join(t.TempDir(), "restored")
	if _, err := app.RestoreBackup(report.Path, restore, key, true); err != nil {
		t.Fatal(err)
	}
	app.Lock()
	if _, err := app.OpenBook(id); err == nil {
		t.Fatal("locked reader access")
	}
	if err := app.UnlockWithRecovery(restore, key); err != nil {
		t.Fatal(err)
	}
	after, err := app.ListBooks()
	if err != nil || len(after) != 5 {
		t.Fatal("restored catalog incomplete")
	}
	exports := t.TempDir()
	for i, b := range after {
		out := filepath.Join(exports, b.ID.String())
		if err := app.ExportOriginal(b.ID.String(), out); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if sha256.Sum256(data) != sha256.Sum256(fixtures[b.Originalname]) {
			t.Fatalf("original mismatch for book %d", i)
		}
		if b.ID.String() == id {
			if b.Position.Progress != 42 || len(b.Shelves) != 1 || len(b.Bookmarks) != 1 {
				t.Fatal("reading metadata lost")
			}
		}
	}
	if err := app.ChangePassword("long-fixture-password", "new-fixture-password"); err != nil {
		t.Fatal(err)
	}
	app.Lock()
	if err := app.Unlock(restore, "long-fixture-password"); err == nil {
		t.Fatal("old password accepted")
	}
	if err := app.Unlock(restore, "new-fixture-password"); err != nil {
		t.Fatal(err)
	}
	app.Lock()
	if err := app.UnlockWithRecovery(restore, key); err != nil {
		t.Fatal("recovery lost after password change")
	}
}
func TestApplicationExclusiveSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library")
	first := NewApp()
	if _, err := first.CreateVault(path, "long-fixture-password"); err != nil {
		t.Fatal(err)
	}
	defer first.Lock()
	second := NewApp()
	if err := second.Unlock(path, "long-fixture-password"); err == nil {
		second.Lock()
		t.Fatal("second writer admitted")
	}
	first.Lock()
	if err := second.Unlock(path, "long-fixture-password"); err != nil {
		t.Fatal(err)
	}
	second.Lock()
}
func TestRestoreWrongCredentialPublishesNothing(t *testing.T) {
	app := NewApp()
	path := filepath.Join(t.TempDir(), "library")
	if _, err := app.CreateVault(path, "long-fixture-password"); err != nil {
		t.Fatal(err)
	}
	defer app.Lock()
	report, err := app.CreateBackup(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if _, err := app.RestoreBackup(report.Path, target, "wrong-password", false); err == nil {
		t.Fatal("wrong credential accepted")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("failed restore published data")
	}
}
