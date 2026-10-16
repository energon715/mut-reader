package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mut/internal/vault"
)

// Verify decrypts every referenced original and cover and checks original hashes.
// It deliberately does not parse untrusted book content during recovery.
func Verify(v *vault.Vault) error {
	c, err := Open(v)
	if err != nil {
		return err
	}
	books, err := c.ListBooks()
	if err != nil {
		return err
	}
	for _, b := range books {
		data, err := v.GetObject(b.FileObjectID)
		if err != nil {
			return fmt.Errorf("original integrity: %w", err)
		}
		hash := sha256.Sum256(data)
		size := len(data)
		clear(data)
		if b.Sha256 != "" && b.Sha256 != hex.EncodeToString(hash[:]) {
			return fmt.Errorf("original checksum mismatch")
		}
		if b.FileSize != size {
			return fmt.Errorf("original size mismatch")
		}
		if b.CoverObjectID != "" {
			cover, err := v.GetObject(b.CoverObjectID)
			if err != nil {
				return fmt.Errorf("cover integrity: %w", err)
			}
			clear(cover)
		}
	}
	return nil
}
