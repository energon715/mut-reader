package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"mut/internal/vault"
	"slices"
	"strings"
	"time"
	"uuid"
)

type Book struct {
	ID            uuid.UUID       `json:"id"`
	Title         string          `json:"title"`
	Author        string          `json:"author"`
	Format        string          `json:"format"`
	FileObjectID  string          `json:"fileobjectid"`
	CoverObjectID string          `json:"coverobjectid"`
	FileSize      int             `json:"filesize"`
	Position      ReadingPosition `json:"position"`
	Shelves       []string        `json:"shelves"`
	Sha256        string          `json:"sha256"`
	Originalname  string          `json:"originalname"`
	Bookmarks     []string        `json:"bookmarks"`
	CreatedAt     time.Time       `json:"createdat"`
	UpdatedAt     time.Time       `json:"updatedat"`
}

type Snapshot struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"createdat"`
	Books     map[string]Book `json:"books"`
	Shelves   []string        `json:"shelves"`
}

type ReadingPosition struct {
	Progress float64   `json:"progress"`
	Locator  string    `json:"locator"`
	UpdateAt time.Time `json:"updateat"`
}

type Catalog struct {
	Vault    *vault.Vault
	Snapshot *Snapshot
}

func (s *Snapshot) ToBytes() ([]byte, error) {
	snapshotBytes, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("cannot return json encoding: %w", err)
	}
	return snapshotBytes, nil
}

func SnapshotFromBytes(data []byte) (*Snapshot, error) {
	var snapshot Snapshot
	err := json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall data: %w", err)
	}
	if snapshot.Books == nil {
		snapshot.Books = make(map[string]Book)
	}
	if snapshot.Shelves == nil {
		snapshot.Shelves = make([]string, 0)
	}
	return &snapshot, nil
}

func Open(v *vault.Vault) (*Catalog, error) {
	lastSnapshot, err := v.GetLastSnapshot()
	var snapshot *Snapshot
	if err != nil {
		if errors.Is(err, vault.ErrNoSnapshot) {
			snapshot = &Snapshot{Version: 0, CreatedAt: time.Now(), Books: make(map[string]Book), Shelves: make([]string, 0)}
		} else {
			return nil, fmt.Errorf("cannot open catalog: %w", err)
		}
	} else {
		snapshot, err = SnapshotFromBytes(lastSnapshot)
		if err != nil {
			return nil, fmt.Errorf("create snapshot from bytes error: %w", err)
		}
	}

	return &Catalog{Vault: v, Snapshot: snapshot}, nil
}

func (c *Catalog) AddBook(book Book) error {
	if book.ID == uuid.Nil() {
		book.ID = uuid.New()
	}
	if c.Snapshot.Books == nil {
		c.Snapshot.Books = make(map[string]Book)
	}
	c.Snapshot.Books[book.ID.String()] = book
	return nil
}

func (c *Catalog) Save() (string, error) {
	if c.Vault == nil {
		return "", fmt.Errorf("vault is not exists")
	}
	if c.Snapshot == nil {
		return "", vault.ErrNoSnapshot
	}
	data, err := c.Snapshot.ToBytes()
	if err != nil {
		return "", fmt.Errorf("snapshot to bytes error: %w", err)
	}
	snapshotID, err := c.Vault.SaveSnapshot(data)
	if err != nil {
		return "", fmt.Errorf("cannot save snapshot: %w", err)
	}
	return snapshotID, nil
}

func (c *Catalog) GetBook(id uuid.UUID) (Book, error) {
	if c.Snapshot == nil || c.Snapshot.Books == nil {
		return Book{}, vault.ErrNoSnapshot
	}

	book, ok := c.Snapshot.Books[id.String()]
	if !ok {
		return Book{}, fmt.Errorf("book not found: %s", id)
	}
	return book, nil
}

func (c *Catalog) ListBooks() ([]Book, error) {
	if c.Snapshot == nil {
		return nil, vault.ErrNoSnapshot
	}
	if c.Snapshot.Books == nil {
		c.Snapshot.Books = make(map[string]Book)
	}
	books := make([]Book, 0, len(c.Snapshot.Books))
	for _, book := range c.Snapshot.Books {
		books = append(books, book)
	}
	return books, nil
}

func (c *Catalog) DeleteBook(id uuid.UUID) error {
	if c.Snapshot == nil || c.Snapshot.Books == nil {
		return vault.ErrNoSnapshot
	}
	_, ok := c.Snapshot.Books[id.String()]
	if !ok {
		return fmt.Errorf("book not found: %s", id)
	}
	delete(c.Snapshot.Books, id.String())
	return nil
}

func (c *Catalog) AddShelf(shelf string) error {
	if c.Snapshot == nil {
		return vault.ErrNoSnapshot
	}
	if c.Snapshot.Shelves == nil {
		c.Snapshot.Shelves = make([]string, 0)
	}
	if shelf == "" {
		return fmt.Errorf("invalid or empty name of shelf")
	}
	if slices.Contains(c.Snapshot.Shelves, shelf) {
		return fmt.Errorf("shelf %s already exists", shelf)
	}
	c.Snapshot.Shelves = append(c.Snapshot.Shelves, shelf)
	return nil
}

func (c *Catalog) ListShelves() ([]string, error) {
	if c.Snapshot == nil {
		return nil, fmt.Errorf("snapshot is not exists")
	}
	if c.Snapshot.Shelves == nil {
		c.Snapshot.Shelves = make([]string, 0)
	}
	return c.Snapshot.Shelves, nil
}

func (c *Catalog) AddBookToShelf(bookID uuid.UUID, shelf string) error {
	book, ok := c.Snapshot.Books[bookID.String()]
	if !ok {
		return fmt.Errorf("book %s not found", bookID)
	}
	if slices.Contains(book.Shelves, shelf) {
		return fmt.Errorf("book %s already on shelf %s", bookID, shelf)
	}
	if !slices.Contains(c.Snapshot.Shelves, shelf) {
		c.Snapshot.Shelves = append(c.Snapshot.Shelves, shelf)
	}
	book.Shelves = append(book.Shelves, shelf)
	c.Snapshot.Books[bookID.String()] = book
	return nil
}

func (c *Catalog) RemoveBookFromShelf(bookID uuid.UUID, shelf string) error {
	if c.Snapshot == nil || c.Snapshot.Books == nil {
		return fmt.Errorf("snapshot not exists")
	}
	book, ok := c.Snapshot.Books[bookID.String()]
	if !ok {
		return fmt.Errorf("book %s not found", bookID)
	}
	idx := slices.Index(book.Shelves, shelf)
	if idx == -1 {
		return fmt.Errorf("book %s not found on shelf %s", bookID, shelf)
	}
	book.Shelves = slices.Delete(book.Shelves, idx, idx+1)
	c.Snapshot.Books[bookID.String()] = book
	return nil
}

func (c *Catalog) DeleteShelf(shelf string) error {
	if c.Snapshot == nil {
		return vault.ErrNoSnapshot
	}
	if !slices.Contains(c.Snapshot.Shelves, shelf) {
		return fmt.Errorf("shelf %s not found", shelf)
	}
	idx := slices.Index(c.Snapshot.Shelves, shelf)
	if idx == -1 {
		return fmt.Errorf("shelf %s not found", shelf)
	}
	c.Snapshot.Shelves = slices.Delete(c.Snapshot.Shelves, idx, idx+1)
	for _, book := range c.Snapshot.Books {
		if slices.Contains(book.Shelves, shelf) {
			c.RemoveBookFromShelf(book.ID, shelf)
		}
	}

	return nil
}

func (c *Catalog) RenameShelf(oldName, newName string) error {
	if c.Snapshot == nil {
		return vault.ErrNoSnapshot
	}
	if !slices.Contains(c.Snapshot.Shelves, oldName) {
		return fmt.Errorf("shelf %s not found", oldName)
	}
	if slices.Contains(c.Snapshot.Shelves, newName) {
		return fmt.Errorf("shelf %s already exists", newName)
	}
	idx := slices.Index(c.Snapshot.Shelves, oldName)
	if idx == -1 {
		return fmt.Errorf("shelf %s not found", oldName)
	}
	c.Snapshot.Shelves[idx] = newName
	for _, book := range c.Snapshot.Books {
		if slices.Contains(book.Shelves, oldName) {
			c.RemoveBookFromShelf(book.ID, oldName)
			c.AddBookToShelf(book.ID, newName)
		}
	}
	return nil
}

func (c *Catalog) FindByHash(hash string) (Book, error) {
	if c.Snapshot == nil {
		return Book{}, vault.ErrNoSnapshot
	}
	for _, book := range c.Snapshot.Books {
		if book.Sha256 == hash {
			return book, nil
		}
	}
	return Book{}, nil
}

func (c *Catalog) Search(query string) []Book {
	if c.Snapshot == nil {
		return nil
	}
	books := make([]Book, 0)
	query = strings.ToLower(query)
	for _, book := range c.Snapshot.Books {
		if strings.Contains(strings.ToLower(book.Title), query) || strings.Contains(strings.ToLower(book.Author), query) {
			books = append(books, book)
		}
	}
	return books
}

func (c *Catalog) ListBooksByShelf(shelf string) ([]Book, error) {
	if c.Snapshot == nil {
		return nil, vault.ErrNoSnapshot
	}
	books := make([]Book, 0)
	for _, book := range c.Snapshot.Books {
		if slices.Contains(book.Shelves, shelf) {
			books = append(books, book)
		}
	}
	return books, nil
}

func (c *Catalog) ListBooksWithoutShelf() []Book {
	if c.Snapshot == nil {
		return nil
	}
	books := make([]Book, 0)
	for _, book := range c.Snapshot.Books {
		if len(book.Shelves) == 0 {
			books = append(books, book)
		}
	}
	return books
}

// func (c *Catalog) ListBookByStatus() // TODO - implement this function

// func (c *Catalog) UpdateReadingPosition(bookID uuid.UUID, progress float64, locator string) error
