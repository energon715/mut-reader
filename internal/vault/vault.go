package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"uuid"
)

type Vault struct {
	path      string
	header    Header
	masterKey []byte
	isLocked  bool
}

func CreateVault(vaulthPath string, password []byte) (*Vault, string, error) {
	if len(password) < 8 {
		return nil, "", fmt.Errorf("password len is less than 8")
	}
	if len(vaulthPath) < 1 {
		return nil, "", fmt.Errorf("vaulthpath is empty")
	}

	jsonPath := filepath.Join(vaulthPath, "vault.json")
	_, err := os.Stat(jsonPath)
	if err == nil {
		return nil, "", fmt.Errorf("vault.json exists at path: %s", vaulthPath)
	}
	if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("file error: %v", err)
	}

	err = os.MkdirAll(filepath.Join(vaulthPath, "objects"), 0700)
	if err != nil {
		return nil, "", fmt.Errorf("make directory error: %v", err)
	}
	err = os.MkdirAll(filepath.Join(vaulthPath, "snapshots"), 0700)
	if err != nil {
		return nil, "", fmt.Errorf("make directory error: %v", err)
	}

	vaultID := uuid.New()
	masterKey, err := generateRandomMasterKey()
	if err != nil {
		return nil, "", fmt.Errorf("masterKey generation failed: %v", err)
	}
	kdf, err := DefaultKDF()
	if err != nil {
		return nil, "", fmt.Errorf("kdf generation error: %v", err)
	}
	wrappingKey := deriveKey(password, kdf)
	keyNonce, wrappedMasterKey, err := encrypt(wrappingKey, masterKey, []byte(vaultID.String()))
	if err != nil {
		return nil, "", fmt.Errorf("Encryprion error: %v", err)
	}
	recoveryBytes, recoveryString, err := generateRecovery()
	if err != nil {
		return nil, "", fmt.Errorf("generate recovery error: %v", err)
	}
	recoveryNonce, cipherRecovery, err := encrypt(recoveryBytes, masterKey, []byte(vaultID.String()+":recovery"))
	if err != nil {
		return nil, "", fmt.Errorf("encrypt recovery error: %v", err)
	}

	payload := append(recoveryNonce, cipherRecovery...)

	header := Header{
		Version:          1,
		VaultID:          vaultID,
		KDF:              kdf,
		WrappedMasterKey: wrappedMasterKey,
		KeyNonce:         keyNonce,
		Recovery:         payload,
	}
	data, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("marshallindent error: %v", err)
	}
	err = os.WriteFile(jsonPath, data, 0600)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write vault.json: %v", err)
	}
	return &Vault{
		path:      vaulthPath,
		header:    header,
		masterKey: masterKey,
		isLocked:  false,
	}, recoveryString, nil
}

func OpenVault(vaultpath string, password []byte) (*Vault, error) {
	if len(vaultpath) < 1 {
		return nil, fmt.Errorf("vaultpath not valid!")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password lenght is less than 8")
	}
	jsonpath := filepath.Join(vaultpath, "vault.json")
	data, err := os.ReadFile(jsonpath)
	if err != nil {
		return nil, fmt.Errorf("Vault not found at this path: %v", err)
	}

	header := Header{}
	err = json.Unmarshal(data, &header)
	if err != nil {
		return nil, fmt.Errorf("unmarshall json error: %v", err)
	}

	wrappingkey := deriveKey(password, header.KDF)
	masterKey, err := decrypt(wrappingkey, header.KeyNonce, header.WrappedMasterKey, []byte(header.VaultID.String()))
	if err != nil {
		return nil, fmt.Errorf("Invalid password or corrupdet vault: %v", err)
	}

	return &Vault{
		path:      vaultpath,
		header:    header,
		masterKey: masterKey,
		isLocked:  false,
	}, nil
}

func OpenVaultWithRecovery(vaultpath string, recoveryKey string) (*Vault, error) {
	if len(vaultpath) < 1 {
		return nil, fmt.Errorf("vaultpath not valid!")
	}
	if recoveryKey == "" {
		return nil, fmt.Errorf("recoveryKey lenght is empty")
	}
	jsonpath := filepath.Join(vaultpath, "vault.json")
	data, err := os.ReadFile(jsonpath)
	if err != nil {
		return nil, fmt.Errorf("Vault not found at this path: %v", err)
	}

	header := Header{}
	err = json.Unmarshal(data, &header)
	if err != nil {
		return nil, fmt.Errorf("unmarshall json error: %v", err)
	}

	if len(header.Recovery) < 40 {
		return nil, fmt.Errorf("Recovery is empty or lenght is less than 40")
	}

	deriveRecovery, err := ParseRecovery(recoveryKey)
	if err != nil {
		return nil, fmt.Errorf("parse recovery error: %v", err)
	}
	recoveryNonce := header.Recovery[:24]
	cipherRecovery := header.Recovery[24:]
	decryptedRecovery, err := decrypt(deriveRecovery, recoveryNonce, cipherRecovery, []byte(header.VaultID.String()+":recovery"))
	if err != nil {
		return nil, fmt.Errorf("decrypt recovery error: %v", err)
	}

	return &Vault{
		path:      vaultpath,
		header:    header,
		masterKey: decryptedRecovery,
		isLocked:  false,
	}, nil

}

func (v *Vault) Lock() {
	v.isLocked = true
	clear(v.masterKey)
	v.masterKey = nil
}
