package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateVault(t *testing.T) {
	tmpDir := t.TempDir()
	vaulthpath := filepath.Join(tmpDir, "test_vault")
	password := []byte("kakashka-vrot123")

	v, recoveryKey, err := CreateVault(vaulthpath, password)
	if err != nil {
		t.Fatalf("create vault failed: %v", err)
	}
	if v == nil {
		t.Fatalf("expected vault object, got nil")
	}
	if len(v.masterKey) != 32 {
		t.Fatalf("lenght of masterkey not right")
	}
	if v.isLocked != false {
		t.Fatalf("created vault is open")
	}
	if v.path != vaulthpath {
		t.Fatalf("path of vault is not right")
	}
	if len(recoveryKey) != 43 {
		t.Fatalf("expected 43-character base64url recovery key, got length %d (%s)", len(recoveryKey), recoveryKey)
	}

	_, err = os.Stat(filepath.Join(vaulthpath, "vault.json"))
	if err != nil {
		t.Fatal("vault.json is not exist, but should be")
	}
	_, err = os.Stat(filepath.Join(vaulthpath, "objects"))
	if err != nil {
		t.Fatal("objects/ is not exist, but should be")
	}
	_, err = os.Stat(filepath.Join(vaulthpath, "snapshots"))
	if err != nil {
		t.Fatal("snapshots/ is not exist, but should be")
	}

	_, _, err = CreateVault(vaulthpath, password)
	if err == nil {
		t.Fatal("expected error when creating vault over existing one")
	}
	shortPasswordPath := filepath.Join(tmpDir, "shortpass-path")
	shortPassword := []byte("kakash")
	_, _, err = CreateVault(shortPasswordPath, shortPassword)
	if err == nil {
		t.Fatalf("expected error for short password")
	}
}

func TestOpenVault(t *testing.T) {
	tmpDir := t.TempDir()
	vaulthpath := filepath.Join(tmpDir, "test_vault")
	password := []byte("kakashka-vrot123")
	createdVault, _, err := CreateVault(vaulthpath, password)
	if err != nil {
		t.Fatalf("creation vault failed: %v", err)
	}

	v, err := OpenVault(vaulthpath, password)
	if err != nil {
		t.Fatalf("Open vault failed: %v", err)
	}
	if v == nil {
		t.Fatalf("vault is nil, expected data")
	}
	if len(v.masterKey) != 32 {
		t.Fatalf("lenght of masterkey not right")
	}
	if v.isLocked != false {
		t.Fatalf("created vault is open")
	}

	if !bytes.Equal(v.masterKey, createdVault.masterKey) {
		t.Fatalf("masterkey mismatch: got %v, want %v", v.masterKey, createdVault.masterKey)
	}

	notvalidpath := filepath.Join(tmpDir, "lolkek")
	_, err = OpenVault(notvalidpath, password)
	if err == nil {
		t.Fatalf("vault opened, but path not exists: %v", err)
	}

	wrongPassword := []byte("kakabyaka-111")
	_, err = OpenVault(vaulthpath, wrongPassword)
	if err == nil {
		t.Fatalf("expected error when opening with wrong password, but got nil")
	}
	v.Lock()

	if !v.isLocked {
		t.Fatalf("expected vault to be locked")
	}
	if v.masterKey != nil {
		t.Fatalf("expected masterkey to be cleared")
	}
}

func TestOpenVaultWithRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "test_recovery_vault")
	password := []byte("my-super-secret-password-1")

	createdVault, recoveryKey, err := CreateVault(vaultPath, password)
	if err != nil {
		t.Fatalf("create vault failed: %v", err)
	}

	// 1. Success recovery unlock
	recoveredVault, err := OpenVaultWithRecovery(vaultPath, recoveryKey)
	if err != nil {
		t.Fatalf("OpenVaultWithRecovery failed: %v", err)
	}
	if recoveredVault == nil {
		t.Fatalf("recovered vault is nil")
	}
	if !bytes.Equal(recoveredVault.masterKey, createdVault.masterKey) {
		t.Fatalf("recovered masterKey mismatch: got %x, want %x", recoveredVault.masterKey, createdVault.masterKey)
	}

	// 2. Put / Get object with recovered vault
	testData := []byte("hello from recovered vault")
	objID, err := recoveredVault.PutObject(testData)
	if err != nil {
		t.Fatalf("put object via recovered vault failed: %v", err)
	}
	gotData, err := recoveredVault.GetObject(objID)
	if err != nil {
		t.Fatalf("get object via recovered vault failed: %v", err)
	}
	if !bytes.Equal(gotData, testData) {
		t.Fatalf("data mismatch: got %s, want %s", gotData, testData)
	}

	// 3. Error on empty recovery key
	_, err = OpenVaultWithRecovery(vaultPath, "")
	if err == nil {
		t.Fatalf("expected error on empty recovery key")
	}

	// 4. Error on wrong recovery key (valid format, but wrong key)
	_, wrongKey, err := generateRecovery()
	if err != nil {
		t.Fatalf("failed to generate dummy recovery key: %v", err)
	}
	_, err = OpenVaultWithRecovery(vaultPath, wrongKey)
	if err == nil {
		t.Fatalf("expected error on wrong recovery key")
	}

	// 5. Error on malformed recovery key (bad base64 or length)
	_, err = OpenVaultWithRecovery(vaultPath, "invalid-key")
	if err == nil {
		t.Fatalf("expected error on malformed recovery key")
	}

	// 6. Error on non-existent path
	_, err = OpenVaultWithRecovery(filepath.Join(tmpDir, "no_such_vault"), recoveryKey)
	if err == nil {
		t.Fatalf("expected error on non-existent path")
	}
}

func TestParseRecovery(t *testing.T) {
	rawBytes, keyString, err := generateRecovery()
	if err != nil {
		t.Fatalf("generateRecovery error: %v", err)
	}

	parsed, err := ParseRecovery(keyString)
	if err != nil {
		t.Fatalf("ParseRecovery error: %v", err)
	}
	if !bytes.Equal(parsed, rawBytes) {
		t.Fatalf("parsed bytes mismatch")
	}

	// whitespace trimming test
	parsedPadded, err := ParseRecovery("  " + keyString + " \n")
	if err != nil {
		t.Fatalf("ParseRecovery with whitespace error: %v", err)
	}
	if !bytes.Equal(parsedPadded, rawBytes) {
		t.Fatalf("parsed bytes with whitespace mismatch")
	}

	// empty input
	_, err = ParseRecovery("")
	if err == nil {
		t.Fatalf("expected error for empty string")
	}

	// wrong length
	_, err = ParseRecovery("c2hvcnQ=")
	if err == nil {
		t.Fatalf("expected error for short key")
	}
}
