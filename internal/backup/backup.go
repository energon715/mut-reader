package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	CreatedAt  time.Time         `json:"createdat"`
	Version    int               `json:"version"`
	SnapshotID string            `json:"snapshotid"`
	VaultID    string            `json:"vault_id"`
	Files      map[string]string `json:"files"`
}

func CreateBackup(vaultPath string, directoryPath string) (*Manifest, error) {
	var manifest Manifest
	err := os.MkdirAll(directoryPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("make directory error: %v", err)
	}
	vaultJSONPath := filepath.Join(vaultPath, "vault.json")
	vaultData, err := os.ReadFile(vaultJSONPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read vault.json error: %v", err)
	}
	if len(vaultData) == 0 {
		return nil, fmt.Errorf("vault.json is empty")
	}
	err = json.Unmarshal(vaultData, &manifest)
	if err != nil {
		return nil, fmt.Errorf("unmarshal vault.json error: %v", err)
	}

	manifest.Files = make(map[string]string)

	err = filepath.WalkDir(vaultPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(directoryPath, relPath)

		hash, err := CreateAndCopy(path, dstPath)
		if err != nil {
			return fmt.Errorf("create and copy file error: %v", err)
		}
		manifest.Files[relPath] = hash
		return nil

	})

	if err != nil {
		return nil, fmt.Errorf("create backup error: %v", err)
	}

	snapshotID, err := os.ReadFile(filepath.Join(vaultPath, "HEAD"))
	if err != nil {
		return nil, fmt.Errorf("cannot read HEAD file error: %v", err)
	}

	manifest.CreatedAt = time.Now().UTC()
	manifest.Version = 1
	manifest.SnapshotID = strings.TrimSpace(string(snapshotID))

	manifestJSON, err := json.MarshalIndent(&manifest, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("cannot marshal files to json error: %v", err)
	}

	manifestJSONPath := filepath.Join(directoryPath, "manifest.json")
	err = os.WriteFile(manifestJSONPath, manifestJSON, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot write manifest.json error: %v", err)
	}

	return &manifest, nil
}

func CreateAndCopy(src, dst string) (string, error) {
	err := os.MkdirAll(filepath.Dir(dst), 0700)
	if err != nil {
		return "", fmt.Errorf("make directory error: %v", err)
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("cannot open src file error: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create dst file error: %v", err)
	}
	defer dstFile.Close()

	h := sha256.New()
	mw := io.MultiWriter(dstFile, h)
	_, err = io.Copy(mw, srcFile)
	if err != nil {
		return "", fmt.Errorf("copy file error: %v", err)
	}

	sha256 := hex.EncodeToString(h.Sum(nil))
	return sha256, nil
}

func VerifyBackup(backupPath string) error {
	manifestJSONPath := filepath.Join(backupPath, "manifest.json")
	manifestJSON, err := os.ReadFile(manifestJSONPath)
	if err != nil {
		return fmt.Errorf("cannot read manifest.json error: %v", err)
	}
	var manifest Manifest
	err = json.Unmarshal(manifestJSON, &manifest)
	if err != nil {
		return fmt.Errorf("cannot unmarshal manifest.json error: %v", err)
	}

	for relPath, expectedHash := range manifest.Files {
		fullpath := filepath.Join(backupPath, relPath)
		hash, err := HashFile(fullpath)
		if err != nil {
			return fmt.Errorf("cannot calculate hash error: %v", err)
		}
		if hash != expectedHash {
			return fmt.Errorf("hash mismatch for file %s: %s != %s", relPath, hash, expectedHash)
		}
	}

	return nil
}

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open file error: %v", err)
	}
	defer file.Close()

	h := sha256.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", fmt.Errorf("cannot calculate hash error: %v", err)
	}
	hash := hex.EncodeToString(h.Sum(nil))
	return hash, nil

}

func RestoreBackup(backupPath, targetVaultPath string) error {
	err := VerifyBackup(backupPath)
	if err != nil {
		return fmt.Errorf("verify backup error: %v", err)
	}
	err = os.MkdirAll(targetVaultPath, 0700)
	if err != nil {
		return fmt.Errorf("make directory error: %v", err)
	}
	manifestJSONPath := filepath.Join(backupPath, "manifest.json")
	manifestJSON, err := os.ReadFile(manifestJSONPath)
	if err != nil {
		return fmt.Errorf("cannot read manifest.json error: %v", err)
	}
	var manifest Manifest
	err = json.Unmarshal(manifestJSON, &manifest)
	if err != nil {
		return fmt.Errorf("cannot unmarshal manifest.json error: %v", err)
	}
	for relPath, _ := range manifest.Files {
		src := filepath.Join(backupPath, relPath)
		dst := filepath.Join(targetVaultPath, relPath)
		_, err = CreateAndCopy(src, dst)
		if err != nil {
			return fmt.Errorf("cannot restore file error: %v", err)
		}
	}
	return nil
}
