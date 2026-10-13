// Package backup copies a bounded vault snapshot into a verified staging
// directory and publishes it without replacing any existing destination.
package backup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mut/internal/storage"
)

type Manifest struct {
	CreatedAt  time.Time         `json:"createdat"`
	Version    int               `json:"version"`
	SnapshotID string            `json:"snapshotid"`
	VaultID    string            `json:"vault_id"`
	Files      map[string]string `json:"files"`
}

func allowed(path string) bool {
	if path == "vault.json" || path == "HEAD" {
		return true
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] == "snapshots" {
		return storage.ValidID(parts[1])
	}
	return len(parts) == 3 && parts[0] == "objects" && storage.ValidID(parts[2]) && parts[1] == parts[2][:2]
}
func stageDestination(source, target string) (string, error) {
	source, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	src, err := filepath.EvalSymlinks(source)
	if err != nil {
		return "", err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if err := storage.EnsureDir(filepath.Dir(target)); err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return "", err
	}
	target = filepath.Join(parent, filepath.Base(target))
	rel, err := filepath.Rel(src, target)
	if err != nil {
		return "", err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		return "", errors.New("destination must be outside source")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		return "", errors.New("destination already exists; choose a new directory")
	}
	return os.MkdirTemp(parent, ".mut-stage-*")
}
func CreateBackup(vaultPath, directoryPath string) (*Manifest, error) {
	stage, err := stageDestination(vaultPath, directoryPath)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	root, err := os.OpenRoot(vaultPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	header, err := readRoot(root, "vault.json", 64<<10)
	if err != nil {
		return nil, err
	}
	head, err := readRoot(root, "HEAD", 128)
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{Version: 1, CreatedAt: time.Now().UTC(), SnapshotID: strings.TrimSpace(string(head)), Files: map[string]string{}}
	var h struct {
		VaultID string `json:"vault_id"`
	}
	if err := json.Unmarshal(header, &h); err != nil {
		return nil, err
	}
	manifest.VaultID = h.VaultID
	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("symlinks are not allowed in a vault backup")
		}
		if entry.IsDir() {
			return nil
		}
		if !allowed(path) {
			return nil
		}
		if len(manifest.Files) >= 100000 {
			return errors.New("backup file count limit exceeded")
		}
		input, err := root.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		hash, err := copyReader(input, filepath.Join(stage, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		manifest.Files[path] = hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	finalHead, err := readRoot(root, "HEAD", 128)
	if err != nil {
		return nil, err
	}
	finalHeader, err := readRoot(root, "vault.json", 64<<10)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(head, finalHead) || !bytes.Equal(header, finalHeader) {
		return nil, errors.New("library changed during backup; retry after closing writers")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := storage.AtomicWrite(filepath.Join(stage, "manifest.json"), data, false); err != nil {
		return nil, err
	}
	if err := VerifyBackup(stage); err != nil {
		return nil, err
	}
	if err := storage.PublishDirectory(stage, directoryPath); err != nil {
		return nil, err
	}
	return manifest, nil
}
func readRoot(root *os.Root, path string, limit int64) ([]byte, error) {
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("invalid backup file")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("backup metadata too large")
	}
	return data, nil
}
func loadManifest(root *os.Root) (*Manifest, error) {
	data, err := readRoot(root, "manifest.json", 64<<20)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 || !storage.ValidID(m.VaultID) || !storage.ValidID(m.SnapshotID) || len(m.Files) < 3 || len(m.Files) > 100000 {
		return nil, errors.New("invalid backup manifest")
	}
	for _, required := range []string{"vault.json", "HEAD", "snapshots/" + m.SnapshotID} {
		if _, ok := m.Files[required]; !ok {
			return nil, errors.New("incomplete backup manifest")
		}
	}
	for path, hash := range m.Files {
		if !allowed(path) || len(hash) != 64 {
			return nil, errors.New("invalid backup path or hash")
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return nil, err
		}
	}
	return &m, nil
}
func verifyRoot(root *os.Root) (*Manifest, error) {
	m, err := loadManifest(root)
	if err != nil {
		return nil, err
	}
	for path, expected := range m.Files {
		f, err := root.Open(path)
		if err != nil {
			return nil, err
		}
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > storage.MaxObjectBytes+40 {
			f.Close()
			return nil, errors.New("invalid backup file")
		}
		hash := sha256.New()
		count, copyErr := io.Copy(hash, io.LimitReader(f, storage.MaxObjectBytes+41))
		err = copyErr
		if count > storage.MaxObjectBytes+40 {
			err = errors.New("backup file size limit exceeded")
		}
		f.Close()
		if err != nil {
			return nil, err
		}
		if hex.EncodeToString(hash.Sum(nil)) != expected {
			return nil, errors.New("backup checksum mismatch")
		}
	}
	head, err := readRoot(root, "HEAD", 128)
	if err != nil || strings.TrimSpace(string(head)) != m.SnapshotID {
		return nil, errors.New("backup HEAD mismatch")
	}
	header, err := readRoot(root, "vault.json", 64<<10)
	if err != nil {
		return nil, err
	}
	var h struct {
		VaultID string `json:"vault_id"`
	}
	if json.Unmarshal(header, &h) != nil || h.VaultID != m.VaultID {
		return nil, errors.New("backup vault ID mismatch")
	}
	return m, nil
}
func VerifyBackup(path string) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	_, err = verifyRoot(root)
	return err
}
func RestoreBackup(path, target string) error { return RestoreValidated(path, target, nil) }

// validate runs against the staged restored vault, before publication.
func RestoreValidated(path, target string, validate func(string) error) error {
	stage, err := stageDestination(path, target)
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	m, err := verifyRoot(root)
	if err != nil {
		return err
	}
	for rel, expected := range m.Files {
		f, err := root.Open(rel)
		if err != nil {
			return err
		}
		actual, err := copyReader(f, filepath.Join(stage, filepath.FromSlash(rel)))
		f.Close()
		if err != nil {
			return err
		}
		if actual != expected {
			return errors.New("backup changed during restoration")
		}
	}
	if validate != nil {
		if err := validate(stage); err != nil {
			return err
		}
	}
	return storage.PublishDirectory(stage, target)
}
func copyReader(input io.Reader, dst string) (string, error) {
	if err := storage.EnsureDir(filepath.Dir(dst)); err != nil {
		return "", err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	count, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(input, storage.MaxObjectBytes+41))
	err = copyErr
	if count > storage.MaxObjectBytes+40 {
		err = errors.New("backup file size limit exceeded")
	}
	if err != nil {
		f.Close()
		return "", err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = storage.SyncDir(filepath.Dir(dst)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func CreateAndCopy(src, dst string) (string, error) {
	input, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer input.Close()
	return copyReader(input, dst)
}
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
