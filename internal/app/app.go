package app

import (
	"context"
	"fmt"
	"mut/internal/catalog"
	"mut/internal/importer"
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

func NewApp() *App {
	return &App{}
}

func (app *App) Startup(ctx context.Context) {
	app.ctx = ctx
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
	app.vaultPath = vaulthPath

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
	app.vaultPath = vaulthPath

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

func (app *App) IsUnlocked() bool {
	return app.vault != nil && app.catalog != nil
}

func (app *App) ListBooks() ([]catalog.Book, error) {
	if app.catalog == nil {
		return nil, fmt.Errorf("catalog is empty")
	}
	bookList, err := app.catalog.ListBooks()
	if err != nil {
		return nil, fmt.Errorf("get list books error: %v", err)
	}

	return bookList, nil
}

func (app *App) ListShelves() ([]string, error) {
	if app.catalog == nil {
		return nil, fmt.Errorf("vault is locked")
	}
	shelves, err := app.catalog.ListShelves()
	if err != nil {
		return nil, fmt.Errorf("get list of shelves error: %v", err)
	}

	return shelves, nil
}

func (app *App) ImportBook(filePath string) error {
	if app.vault == nil || app.catalog == nil {
		return vault.ErrVaultLocked
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file error: %v", err)
	}
	fileName := filepath.Base(filePath)
	err = importer.ImportFile(fileName, app.vault, app.catalog, data, false)
	if err != nil {
		return fmt.Errorf("Import file error: %v", err)
	}

	return nil

}
