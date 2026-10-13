package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreDoesNotOverwrite(t *testing.T) {
	source, _, _, _ := setupTestVault(t)
	copy := filepath.Join(t.TempDir(), "copy")
	if _, err := CreateBackup(source, copy); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	sentinel := filepath.Join(target, "keep.txt")
	os.WriteFile(sentinel, []byte("keep"), 0600)
	if err := RestoreBackup(copy, target); err == nil {
		t.Fatal("existing directory overwritten")
	}
	b, _ := os.ReadFile(sentinel)
	if string(b) != "keep" {
		t.Fatal("user data changed")
	}
}
func TestManifestPathTraversalRejected(t *testing.T) {
	source, _, _, _ := setupTestVault(t)
	copy := filepath.Join(t.TempDir(), "copy")
	m, err := CreateBackup(source, copy)
	if err != nil {
		t.Fatal(err)
	}
	m.Files["../../outside"] = m.Files["HEAD"]
	b, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(copy, "manifest.json"), b, 0600)
	if err := VerifyBackup(copy); err == nil {
		t.Fatal("traversal manifest accepted")
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := RestoreBackup(copy, target); err == nil {
		t.Fatal("traversal restored")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("failed restore published a directory")
	}
}
func TestSymlinkEscapeRejected(t *testing.T) {
	source, _, _, _ := setupTestVault(t)
	copy := filepath.Join(t.TempDir(), "copy")
	m, err := CreateBackup(source, copy)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	b, _ := os.ReadFile(filepath.Join(copy, "HEAD"))
	os.WriteFile(outside, b, 0600)
	os.Remove(filepath.Join(copy, "HEAD"))
	os.Symlink(outside, filepath.Join(copy, "HEAD"))
	if err := VerifyBackup(copy); err == nil {
		t.Fatal("external symlink accepted")
	}
	_ = m
}
func TestBackupCannotBeInsideVault(t *testing.T) {
	source, _, _, _ := setupTestVault(t)
	if _, err := CreateBackup(source, filepath.Join(source, "backup")); err == nil {
		t.Fatal("recursive backup accepted")
	}
}
