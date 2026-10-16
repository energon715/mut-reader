package vault

import (
	"errors"
	"fmt"
	"mut/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"uuid"
)

var ErrNoSnapshot = errors.New("cannot find snapshot")

func (v *Vault) SaveSnapshot(data []byte) (snapshotID string, err error) {
	if int64(len(data)) > storage.MaxCatalogBytes {
		return "", fmt.Errorf("catalog too large")
	}
	if v.isLocked {
		return "", ErrVaultLocked
	}
	snapshotID = uuid.New().String()

	snapshotKey, err := deriveEntityKey(v.masterKey, v.header.VaultID.String(), "snapshot", snapshotID)
	if err != nil {
		return "", fmt.Errorf("derive snapshotKey error: %v", err)
	}
	aad := []byte("snapshot:" + snapshotID)

	nonce, ciphertext, err := encrypt(snapshotKey, data, aad)
	if err != nil {
		return "", fmt.Errorf("encryptin snapshot error: %v", err)
	}
	payload := append(nonce, ciphertext...)

	headPath := filepath.Join(v.path, "HEAD")

	dir := filepath.Join(v.path, "snapshots")
	err = storage.EnsureDir(dir)
	if err != nil {
		return "", fmt.Errorf("make directory error: %v", err)
	}

	snapshotpath := filepath.Join(dir, snapshotID)
	err = storage.AtomicWrite(snapshotpath, payload, false)
	if err != nil {
		return "", fmt.Errorf("write snapshot error: %v", err)
	}

	err = storage.AtomicWrite(headPath, []byte(snapshotID+"\n"), true)
	if err != nil {
		return "", fmt.Errorf("commit HEAD: %w", err)
	}

	return snapshotID, nil

}

func (v *Vault) GetSnapshot(snapshotID string) ([]byte, error) {
	if !storage.ValidID(snapshotID) {
		return nil, fmt.Errorf("lenght of snapshotID less than 2")
	}
	if v.isLocked {
		return nil, ErrVaultLocked
	}
	snapshotPath := filepath.Join(v.path, "snapshots", snapshotID)
	payload, err := storage.ReadFile(snapshotPath, storage.MaxCatalogBytes+40)
	if err != nil {
		return nil, fmt.Errorf("cannot read snapshot file: %v", err)
	}
	if len(payload) < 40 {
		return nil, fmt.Errorf("invalid length of snapshot")
	}
	nonce := payload[:24]
	ciphertext := payload[24:]

	snapshotKey, err := deriveEntityKey(v.masterKey, v.header.VaultID.String(), "snapshot", snapshotID)
	if err != nil {
		return nil, fmt.Errorf("derive key error: %v", err)
	}
	aad := []byte("snapshot:" + snapshotID)

	data, err := decrypt(snapshotKey, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption error: %v", err)
	}

	return data, nil
}

func (v *Vault) GetLastSnapshot() ([]byte, error) {
	snapshotID, err := v.GetHead()
	if err != nil {
		return nil, fmt.Errorf("cannot get snapshotID: %w", err)
	}
	return v.GetSnapshot(snapshotID)
}

func (v *Vault) GetHead() (string, error) {
	if v.isLocked {
		return "", ErrVaultLocked
	}
	head, err := storage.ReadFile(filepath.Join(v.path, "HEAD"), 128)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoSnapshot
		}
		return "", fmt.Errorf("read HEAD error: %v", err)
	}
	snapshotID := strings.TrimSpace(string(head))
	if snapshotID == "" {
		return "", ErrNoSnapshot
	}
	if !storage.ValidID(snapshotID) {
		return "", fmt.Errorf("invalid snapshot ID")
	}
	return snapshotID, nil
}
