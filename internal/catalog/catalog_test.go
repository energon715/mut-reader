package catalog

import (
	"mut/internal/vault"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "test_vaulth")
	password := []byte("kakash-kavrot")

	vault, err := vault.CreateVault(vaultPath, password)
	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	c, err := Open(vault)
	if err != nil {
		t.Fatalf("Open cataalog error: %v", err)
	}
	if c == nil {
		t.Fatalf("catalog is empty, expected not nil")
	}
	if len(c.Snapshot.Books) != 0 {
		t.Fatalf("catalog books count not zero, expected 0")
	}

	book := Book{
		Title:  "TEst",
		Author: "Dronov Kirill",
	}
	err = c.AddBook(book)
	if err != nil {
		t.Fatalf("add book error: %v", err)
	}
	savedID, err := c.Save()
	if err != nil {
		t.Fatalf("save catalog error: %v", err)
	}
	headID, err := vault.GetHead()
	if err != nil {
		t.Fatalf("get head error: %v", err)
	}
	if headID != savedID {
		t.Fatalf("head not equal saved, expected %s, got %s", savedID, headID)
	}
	c2, err := Open(vault)
	if err != nil {
		t.Fatalf("open catalog 2 error: %v", err)
	}
	books, err := c2.ListBooks()
	if err != nil {
		t.Fatalf("list books error: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("books count not one, expected 1, got %d", len(books))
	}
	if books[0].Title != "TEst" || books[0].Author != "Dronov Kirill" {
		t.Fatalf("book not equal, expected %+v, got %+v", book, books[0])
	}
	savedBook, err := c2.GetBook(books[0].ID)
	if err != nil {
		t.Fatalf("get book error: %v", err)
	}
	if savedBook.Title != "TEst" {
		t.Fatalf("title mismatch")
	}
	err = c.DeleteBook(books[0].ID)
	if err != nil {
		t.Fatalf("delete book error: %v", err)
	}
	_, err = c.GetBook(books[0].ID)
	if err == nil {
		t.Fatalf("book is not exist, but geted")
	}

}
