package app

import (
	"context"
	"fmt"
	"mut/internal/catalog"
	"mut/internal/vault"
	"os"
	"path/filepath"
)

type App struct {
	ctx       context.Context
	vault     *vault.Vault
	catalog   *catalog.Catalog
	vaultPath string
}

func (app *App) IsVaultConfigurated() bool {

	if app.vaultPath == "" {
		app.vaultPath = defaultVaultPath()
	}

	vaultPath := filepath.Join(app.vaultPath, "vault.json")

	info, err := os.Stat(vaultPath)
	if err == nil {
		return !info.IsDir()
	} else {
		return false
	}
}

func (app *App) CreateVault(vaultPath string, password string) (recoveryKey string, err error) {

	v, recoveryKey, err := vault.CreateVault(vaultPath, []byte(password))
	if err != nil {
		return "", fmt.Errorf("create vault error; %v", err)
	}

	c, err := catalog.Open(v)
	if err != nil {
		return "", fmt.Errorf("catalog open error: %v", err)
	}
	_, err = c.Save()
	if err != nil {
		return "", fmt.Errorf("save catalog error: %v", err)
	}

	app.vault = v
	app.catalog = c
	app.vaultPath = vaultPath

	return recoveryKey, nil

}

func (app *App) Unlock(vaulthPath string, password string) error {
	v, err := vault.OpenVault(vaulthPath, []byte(password))
	if err != nil {
		return fmt.Errorf("open vault error: %v", err)
	}
	c, err := catalog.Open(v)
	if err != nil {
		return fmt.Errorf("open catalog error: %v", err)
	}

	app.catalog = c
	app.vault = v

	return nil
}

func (app *App) UnlockWithRecovery(vaulthPath string, recoveryKey string) error {
	v, err := vault.OpenVaultWithRecovery(vaulthPath, recoveryKey)
	if err != nil {
		return fmt.Errorf("open vault with recovery error: %v", err)
	}
	c, err := catalog.Open(v)
	if err != nil {
		return fmt.Errorf("open catalog error: %v", err)
	}

	app.catalog = c
	app.vault = v

	return nil
}

func (app *App) Lock() {
	if app.vault != nil {
		app.vault.Lock()
	}
	app.vault = nil
	app.catalog = nil
}

func defaultVaultPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	path := filepath.Join(homeDir, ".local", "share", "mut", "vault")

	return path
}
