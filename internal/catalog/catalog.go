package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"mut/internal/vault"
	"slices"
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
