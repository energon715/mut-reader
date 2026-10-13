package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mut/internal/catalog"
	"mut/internal/importer"
	"mut/internal/vault"
)

func setTestStdin(input string) {
	inReader = strings.NewReader(input)
	stdinScanner = nil
}

func createTestVaultWithBook(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "test_vault")
	password := []byte("vaultctl-test-password-1")

	v, recoveryKey, err := vault.CreateVault(vaultPath, password)
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	c, err := catalog.Open(v)
	if err != nil {
		t.Fatalf("catalog.Open failed: %v", err)
	}

	pdfContent := []byte("%PDF-1.4\n<< /Title (Vaultctl Manual) /Author (Mut Team) >>\n%%EOF")
	err = importer.ImportFile("manual.pdf", v, c, pdfContent, false)
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}

	return vaultPath, recoveryKey
}

func TestVaultctlBackupVerifyRestore(t *testing.T) {
	vaultPath, recoveryKey := createTestVaultWithBook(t)
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")
	restoredVault := filepath.Join(tmpDir, "restored_vault")

	// 1. Test backup
	err := runBackup([]string{vaultPath, backupDir})
	if err != nil {
		t.Fatalf("runBackup failed: %v", err)
	}

	// 2. Test verify
	err = runVerify([]string{backupDir})
	if err != nil {
		t.Fatalf("runVerify failed: %v", err)
	}

	// 3. Test authenticated restore
	setTestStdin(recoveryKey + "\n")
	err = runRestore([]string{backupDir, restoredVault})
	if err != nil {
		t.Fatalf("runRestore failed: %v", err)
	}

	// Verify restored vault is functional
	v, err := vault.OpenVault(restoredVault, []byte("vaultctl-test-password-1"))
	if err != nil {
		t.Fatalf("OpenVault on restored vault failed: %v", err)
	}
	c, err := catalog.Open(v)
	if err != nil {
		t.Fatalf("catalog.Open on restored vault failed: %v", err)
	}
	books, err := c.ListBooks()
	if err != nil || len(books) != 1 {
		t.Fatalf("expected 1 book in restored vault, got %d (err: %v)", len(books), err)
	}
}

func TestVaultctlCheckAndExport(t *testing.T) {
	vaultPath, recoveryKey := createTestVaultWithBook(t)
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "exported")

	// Mock stdin for check
	setTestStdin(recoveryKey + "\n")
	err := runCheck([]string{vaultPath})
	if err != nil {
		t.Fatalf("runCheck failed: %v", err)
	}

	// Mock stdin for export
	setTestStdin(recoveryKey + "\n")
	err = runExport([]string{vaultPath, outputDir})
	if err != nil {
		t.Fatalf("runExport failed: %v", err)
	}

	// Verify exported book file
	exportedFiles, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read export dir: %v", err)
	}
	if len(exportedFiles) != 1 {
		t.Fatalf("expected 1 exported file, got %d", len(exportedFiles))
	}
	data, err := os.ReadFile(filepath.Join(outputDir, exportedFiles[0].Name()))
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) {
		t.Fatalf("exported file content corrupted")
	}
}

func TestVaultctlPurge(t *testing.T) {
	vaultPath, recoveryKey := createTestVaultWithBook(t)

	// Mock user typing recovery key, then confirming "PURGE"
	setTestStdin(recoveryKey + "\nPURGE\n")
	err := runPurge([]string{vaultPath})
	if err != nil {
		t.Fatalf("runPurge failed: %v", err)
	}

	// Verify directory was completely destroyed
	if _, err := os.Stat(vaultPath); !os.IsNotExist(err) {
		t.Fatalf("expected vault directory to be deleted, but it still exists")
	}
}

func TestRestoreWrongSecretDoesNotPublish(t *testing.T) {
	path, _ := createTestVaultWithBook(t)
	backupPath := filepath.Join(t.TempDir(), "backup")
	if err := runBackup([]string{path, backupPath}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "restored")
	setTestStdin("wrong-password\n")
	if err := runRestore([]string{backupPath, target, "--password"}); err == nil {
		t.Fatal("wrong secret restored")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("failed restore published a directory")
	}
}

func TestExportPreservesExisting(t *testing.T) {
	path, key := createTestVaultWithBook(t)
	out := t.TempDir()
	original := filepath.Join(out, "manual.pdf")
	if err := os.WriteFile(original, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	setTestStdin(key + "\n")
	if err := runExport([]string{path, out}); err == nil {
		t.Fatal("partial export reported success")
	}
	data, _ := os.ReadFile(original)
	if string(data) != "keep me" {
		t.Fatal("existing data overwritten")
	}
}
