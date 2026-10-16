package vault

import (
	"errors"
	"fmt"
	"mut/internal/storage"
	"path/filepath"
	"uuid"
)

var ErrVaultLocked = errors.New("vault is locked")

func (v *Vault) PutObject(data []byte) (objectID string, err error) {
	if int64(len(data)) > storage.MaxObjectBytes {
		return "", fmt.Errorf("object exceeds 256 MiB limit")
	}
	if v.isLocked {
		return "", ErrVaultLocked
	}
	objectID = uuid.New().String()
	objKey, err := deriveEntityKey(v.masterKey, v.header.VaultID.String(), "object", objectID)
	if err != nil {
		return "", fmt.Errorf("derive obj key error: %v", err)
	}
	aad := []byte("object:" + objectID)
	nonce, ciphertext, err := encrypt(objKey, data, aad)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %v", err)
	}
	payload := append(nonce, ciphertext...)
	dir := filepath.Join(v.path, "objects", objectID[:2])
	err = storage.EnsureDir(dir)
	if err != nil {
		return "", fmt.Errorf("make directory error: %v", err)
	}
	objectpath := filepath.Join(dir, objectID)
	err = storage.AtomicWrite(objectpath, payload, false)
	if err != nil {
		return "", fmt.Errorf("make directory error: %v", err)
	}
	return objectID, nil
}

func (v *Vault) GetObject(objectID string) ([]byte, error) {
	if !storage.ValidID(objectID) {
		return nil, fmt.Errorf("lenght of objectID less than 2")
	}
	if v.isLocked {
		return nil, ErrVaultLocked
	}
	objectPath := filepath.Join(v.path, "objects", objectID[:2], objectID)
	payload, err := storage.ReadFile(objectPath, storage.MaxObjectBytes+40)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %v", err)
	}
	if len(payload) < 40 {
		return nil, fmt.Errorf("invalin lenght of object: %v", err)
	}
	nonce := payload[:24]
	ciphertext := payload[24:]

	objectKey, err := deriveEntityKey(v.masterKey, v.header.VaultID.String(), "object", objectID)
	if err != nil {
		return nil, fmt.Errorf("derive key error: %v", err)
	}
	aad := []byte("object:" + objectID)

	data, err := decrypt(objectKey, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption error: %v", err)
	}

	return data, nil
}
