package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"mut/internal/vault"
)

func setupTestVault(t *testing.T) (string, []byte, string, []string) {
	t.Helper()
	vaultDir := filepath.Join(t.TempDir(), "test_vault")
	password := []byte("strong-backup-password-123")

	v, err := vault.CreateVault(vaultDir, password)
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	objData1 := []byte("book content 1: the quick brown fox")
	objData2 := []byte("book content 2: jumps over the lazy dog")

	objID1, err := v.PutObject(objData1)
	if err != nil {
		t.Fatalf("PutObject 1 failed: %v", err)
	}

	objID2, err := v.PutObject(objData2)
	if err != nil {
		t.Fatalf("PutObject 2 failed: %v", err)
	}

	snapshotData := []byte("catalog snapshot with 2 books")
	snapID, err := v.SaveSnapshot(snapshotData)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	return vaultDir, password, snapID, []string{objID1, objID2}
}

func TestCreateAndVerifyBackup(t *testing.T) {
	vaultDir, _, snapID, _ := setupTestVault(t)
	backupDir := filepath.Join(t.TempDir(), "backup_dir")

	manifest, err := CreateBackup(vaultDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if manifest == nil {
		t.Fatal("expected non-nil manifest")
	}

	if manifest.SnapshotID != snapID {
		t.Fatalf("expected SnapshotID %q, got %q", snapID, manifest.SnapshotID)
	}

	if manifest.VaultID == "" {
		t.Fatal("expected non-empty VaultID in manifest")
	}

	if manifest.Version != 1 {
		t.Fatalf("expected Version 1, got %d", manifest.Version)
	}

	if len(manifest.Files) == 0 {
		t.Fatal("expected manifest to contain files")
	}

	// Проверяем, что ключевые файлы хранилища учтены в манифесте
	if _, ok := manifest.Files["vault.json"]; !ok {
		t.Error("manifest is missing vault.json")
	}
	if _, ok := manifest.Files["HEAD"]; !ok {
		t.Error("manifest is missing HEAD")
	}

	// Проверяем наличие manifest.json на диске
	manifestPath := filepath.Join(backupDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest.json was not created on disk: %v", err)
	}

	// Проверяем целостность бэкапа
	if err := VerifyBackup(backupDir); err != nil {
		t.Fatalf("VerifyBackup failed on valid backup: %v", err)
	}
}

func TestVerifyBackupCorruptedFile(t *testing.T) {
	vaultDir, _, _, _ := setupTestVault(t)
	backupDir := filepath.Join(t.TempDir(), "backup_dir")

	_, err := CreateBackup(vaultDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Намеренно портим один файл (например, дописываем байт в HEAD)
	headPath := filepath.Join(backupDir, "HEAD")
	f, err := os.OpenFile(headPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("open HEAD failed: %v", err)
	}
	if _, err := f.Write([]byte("corrupted")); err != nil {
		t.Fatalf("corrupt HEAD failed: %v", err)
	}
	_ = f.Close()

	err = VerifyBackup(backupDir)
	if err == nil {
		t.Fatal("expected VerifyBackup to fail for corrupted file, but got nil")
	}
}

func TestVerifyBackupMissingFile(t *testing.T) {
	vaultDir, _, _, _ := setupTestVault(t)
	backupDir := filepath.Join(t.TempDir(), "backup_dir")

	_, err := CreateBackup(vaultDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Удаляем vault.json из бэкапа
	vaultJSON := filepath.Join(backupDir, "vault.json")
	if err := os.Remove(vaultJSON); err != nil {
		t.Fatalf("remove vault.json failed: %v", err)
	}

	err = VerifyBackup(backupDir)
	if err == nil {
		t.Fatal("expected VerifyBackup to fail for missing file, but got nil")
	}
}

func TestRestoreBackup(t *testing.T) {
	vaultDir, password, snapID, objIDs := setupTestVault(t)
	backupDir := filepath.Join(t.TempDir(), "backup_dir")
	restoredDir := filepath.Join(t.TempDir(), "restored_vault")

	_, err := CreateBackup(vaultDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	err = RestoreBackup(backupDir, restoredDir)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Убеждаемся, что manifest.json НЕ был скопирован в восстановленное хранилище
	if _, err := os.Stat(filepath.Join(restoredDir, "manifest.json")); !os.IsNotExist(err) {
		t.Error("manifest.json should not exist in restored vault")
	}

	// Открываем восстановленное хранилище оригинальным паролем
	restoredVault, err := vault.OpenVault(restoredDir, password)
	if err != nil {
		t.Fatalf("OpenVault on restored vault failed: %v", err)
	}

	// Проверяем последний снепшот
	head, err := restoredVault.GetHead()
	if err != nil {
		t.Fatalf("GetHead failed on restored vault: %v", err)
	}
	if head != snapID {
		t.Fatalf("expected head %q, got %q", snapID, head)
	}

	lastSnapData, err := restoredVault.GetLastSnapshot()
	if err != nil {
		t.Fatalf("GetLastSnapshot failed: %v", err)
	}
	if !bytes.Equal(lastSnapData, []byte("catalog snapshot with 2 books")) {
		t.Fatalf("snapshot data mismatch, got %q", string(lastSnapData))
	}

	// Проверяем чтение объектов
	obj1Data, err := restoredVault.GetObject(objIDs[0])
	if err != nil {
		t.Fatalf("GetObject 1 failed: %v", err)
	}
	if !bytes.Equal(obj1Data, []byte("book content 1: the quick brown fox")) {
		t.Fatalf("object 1 data mismatch: %q", string(obj1Data))
	}

	obj2Data, err := restoredVault.GetObject(objIDs[1])
	if err != nil {
		t.Fatalf("GetObject 2 failed: %v", err)
	}
	if !bytes.Equal(obj2Data, []byte("book content 2: jumps over the lazy dog")) {
		t.Fatalf("object 2 data mismatch: %q", string(obj2Data))
	}
}

func TestRestoreCorruptedBackupFails(t *testing.T) {
	vaultDir, _, _, _ := setupTestVault(t)
	backupDir := filepath.Join(t.TempDir(), "backup_dir")
	restoredDir := filepath.Join(t.TempDir(), "restored_vault")

	_, err := CreateBackup(vaultDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Портим manifest.json или файл в бэкапе
	vaultJSON := filepath.Join(backupDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte("tampered data"), 0600); err != nil {
		t.Fatalf("tampering failed: %v", err)
	}

	err = RestoreBackup(backupDir, restoredDir)
	if err == nil {
		t.Fatal("expected RestoreBackup to fail on corrupted backup, but got nil")
	}
}
