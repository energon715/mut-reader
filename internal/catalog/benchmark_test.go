package catalog

import (
	"fmt"
	"mut/internal/vault"
	"path/filepath"
	"testing"
	"uuid"
)

// This isolates catalog cost with synthetic metadata, not real book decoding.
func BenchmarkEncryptedCatalogOpen(b *testing.B) {
	for _, count := range []int{1, 100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			v, _, err := vault.CreateVault(filepath.Join(b.TempDir(), "vault"), []byte("benchmark-password"))
			if err != nil {
				b.Fatal(err)
			}
			defer v.Lock()
			c, err := Open(v)
			if err != nil {
				b.Fatal(err)
			}
			for i := 0; i < count; i++ {
				id := uuid.New()
				_ = c.AddBook(Book{ID: id, Title: fmt.Sprintf("Synthetic book %d", i), Author: "Synthetic author", Format: "EPUB", FileObjectID: uuid.New().String(), Bookmarks: []string{"v1:text:0:0"}, Shelves: []string{"Shelf"}})
			}
			if _, err := c.Save(); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				catalog, err := Open(v)
				if err != nil {
					b.Fatal(err)
				}
				books, err := catalog.ListBooks()
				if err != nil || len(books) != count {
					b.Fatal("catalog count mismatch")
				}
			}
		})
	}
}
