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

func TestShelvesDeleteAndRename(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault_shelves")
	v, err := vault.CreateVault(vaultPath, []byte("test-password-123"))
	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	c, err := Open(v)
	if err != nil {
		t.Fatalf("open catalog error: %v", err)
	}

	err = c.AddShelf("Sci-Fi")
	if err != nil {
		t.Fatalf("add shelf error: %v", err)
	}

	book := Book{
		Title:  "Dune",
		Author: "Frank Herbert",
	}
	err = c.AddBook(book)
	if err != nil {
		t.Fatalf("add book error: %v", err)
	}
	books, _ := c.ListBooks()
	bookID := books[0].ID

	err = c.AddBookToShelf(bookID, "Sci-Fi")
	if err != nil {
		t.Fatalf("add book to shelf error: %v", err)
	}

	// Rename shelf
	err = c.RenameShelf("Sci-Fi", "Science Fiction")
	if err != nil {
		t.Fatalf("rename shelf error: %v", err)
	}
	updatedBook, _ := c.GetBook(bookID)
	if len(updatedBook.Shelves) != 1 || updatedBook.Shelves[0] != "Science Fiction" {
		t.Fatalf("expected renamed shelf on book, got %v", updatedBook.Shelves)
	}

	// Delete shelf
	err = c.DeleteShelf("Science Fiction")
	if err != nil {
		t.Fatalf("delete shelf error: %v", err)
	}
	updatedBook, _ = c.GetBook(bookID)
	if len(updatedBook.Shelves) != 0 {
		t.Fatalf("expected shelf removed from book, got %v", updatedBook.Shelves)
	}
	shelves, _ := c.ListShelves()
	if len(shelves) != 0 {
		t.Fatalf("expected shelves list to be empty, got %v", shelves)
	}
}

func TestSearch(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault_search")
	v, err := vault.CreateVault(vaultPath, []byte("test-password-123"))
	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	c, err := Open(v)
	if err != nil {
		t.Fatalf("open catalog error: %v", err)
	}

	_ = c.AddBook(Book{Title: "Master and Margarita", Author: "Mikhail Bulgakov"})
	_ = c.AddBook(Book{Title: "Heart of a Dog", Author: "Mikhail Bulgakov"})
	_ = c.AddBook(Book{Title: "The Idiot", Author: "Fyodor Dostoevsky"})

	// Search by author
	res := c.Search("bulgakov")
	if len(res) != 2 {
		t.Fatalf("expected 2 books for Bulgakov, got %d", len(res))
	}

	// Search by title (case insensitive)
	res = c.Search("IDIOT")
	if len(res) != 1 || res[0].Title != "The Idiot" {
		t.Fatalf("expected 'The Idiot', got %+v", res)
	}

	// Search nonexistent
	res = c.Search("nonexistent")
	if len(res) != 0 {
		t.Fatalf("expected 0 books, got %d", len(res))
	}
}

func TestFindByHash(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault_hash")
	v, err := vault.CreateVault(vaultPath, []byte("test-password-123"))
	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	c, err := Open(v)
	if err != nil {
		t.Fatalf("open catalog error: %v", err)
	}

	testHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	_ = c.AddBook(Book{
		Title:  "Sample Book",
		Sha256: testHash,
	})

	foundBook, found, err := c.FindByHash(testHash)
	if err != nil {
		t.Fatalf("find by hash error: %v", err)
	}
	if !found || foundBook.Title != "Sample Book" {
		t.Fatalf("expected to find Sample Book, got found=%v, book=%+v", found, foundBook)
	}

	_, found, err = c.FindByHash("unknown-hash")
	if err != nil {
		t.Fatalf("find by hash error: %v", err)
	}
	if found {
		t.Fatalf("expected not found for unknown hash")
	}
}

func TestListBooksByShelfAndWithoutShelf(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault_shelf_lists")
	v, err := vault.CreateVault(vaultPath, []byte("test-password-123"))
	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	c, err := Open(v)
	if err != nil {
		t.Fatalf("open catalog error: %v", err)
	}

	_ = c.AddShelf("History")

	b1 := Book{Title: "Book on Shelf", Shelves: []string{"History"}}
	b2 := Book{Title: "Book without Shelf", Shelves: []string{}}

	_ = c.AddBook(b1)
	_ = c.AddBook(b2)

	historyBooks, err := c.ListBooksByShelf("History")
	if err != nil {
		t.Fatalf("list by shelf error: %v", err)
	}
	if len(historyBooks) != 1 || historyBooks[0].Title != "Book on Shelf" {
		t.Fatalf("expected 1 history book, got %+v", historyBooks)
	}

	noShelfBooks := c.ListBooksWithoutShelf()
	if len(noShelfBooks) != 1 || noShelfBooks[0].Title != "Book without Shelf" {
		t.Fatalf("expected 1 book without shelf, got %+v", noShelfBooks)
	}
}
