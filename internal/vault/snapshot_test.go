package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "snapshot_test")
	password := []byte("kakash-kavrot")
	v, _, err := CreateVault(snapshotPath, password)

	if err != nil {
		t.Fatalf("create vault error: %v", err)
	}
	if v.isLocked {
		t.Fatalf("vault is locked, expected open")
	}

	_, err = v.GetHead()
	if err == nil {
		t.Fatalf("vault is empty, expected error")
	}
	_, err = v.GetLastSnapshot()
	if err == nil {
		t.Fatalf("vault is empty, expected error")
	}

	data := []byte("random-data")
	snapshotID, err := v.SaveSnapshot(data)
	if err != nil {
		t.Fatalf("Save snapshot error: %v", err)
	}
	getedLastSnapshot, err := v.GetLastSnapshot()
	if err != nil {
		t.Fatalf("get last snapshot error: %v", err)
	}
	rawOnDisk, err := os.ReadFile(filepath.Join(snapshotPath, "snapshots", snapshotID))
	if err != nil {
		t.Fatalf("read raw snapshot error: %v", err)
	}
	if !bytes.Equal(getedLastSnapshot, data) {
		t.Fatalf("expected decrypted data to match original")
	}

	if bytes.Equal(getedLastSnapshot, rawOnDisk) {
		t.Fatalf("writed snapshot not encrypted")
	}
	expectedSnapshotID, err := v.GetHead()
	if err != nil {
		t.Fatalf("Get snapshot id Error: %v", err)
	}
	if expectedSnapshotID != snapshotID {
		t.Fatalf("snapshotID in vault not right!")
	}
	headBytes, err := os.ReadFile(filepath.Join(snapshotPath, "HEAD"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("HEAD is not exists: %v", err)
		}
	}
	if strings.TrimSpace(string(headBytes)) != snapshotID {
		t.Fatalf("writed snapShotID is not right!")
	}
	v.Lock()
	_, err = v.GetSnapshot(snapshotID)
	if err == nil {
		t.Fatalf("vault id locked, but we can get snapshot")
	}
}
