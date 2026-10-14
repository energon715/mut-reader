// Package storage provides bounded reads and durable, atomic file commits.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"uuid"

	"golang.org/x/sys/unix"
)

const MaxObjectBytes int64 = 256 << 20
const MaxCatalogBytes int64 = 64 << 20

func ReadFile(path string, limit int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("file is not regular or exceeds the size limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		clear(data)
		return nil, errors.New("file exceeds the size limit")
	}
	return data, nil
}
func SyncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for p := dir; ; p = filepath.Dir(p) {
		if err := SyncDir(p); err != nil {
			return err
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}
func AtomicWrite(path string, data []byte, replace bool) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".mut-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if replace {
		err = os.Rename(f.Name(), path)
	} else {
		err = os.Link(f.Name(), path)
	}
	if err != nil {
		return err
	}
	return SyncDir(filepath.Dir(path))
}
func PublishDirectory(stage, target string) error {
	if err := SyncDir(stage); err != nil {
		return err
	}
	if err := unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, target, unix.RENAME_NOREPLACE); err != nil {
		return fmt.Errorf("publish directory without overwriting: %w", err)
	}
	return SyncDir(filepath.Dir(target))
}
func ValidID(id string) bool {
	u, err := uuid.Parse(id)
	return err == nil && u != uuid.Nil() && u.String() == id
}
