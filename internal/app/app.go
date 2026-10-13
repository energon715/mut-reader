package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"uuid"

	"mut/internal/backup"
	"mut/internal/catalog"
	"mut/internal/importer"
	"mut/internal/readers"
	"mut/internal/storage"
	"mut/internal/vault"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/unix"
)

type App struct {
	ctx          context.Context
	mu           sync.Mutex
	vault        *vault.Vault
	catalog      *catalog.Catalog
	vaultPath    string
	sessionLock  *os.File
	lastActivity time.Time
	autoLock     time.Duration
}

func NewApp() *App { return &App{autoLock: 20 * time.Minute, lastActivity: time.Now()} }
func (app *App) Startup(ctx context.Context) {
	app.ctx = ctx
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app.mu.Lock()
				if app.vault != nil && app.autoLock > 0 && time.Since(app.lastActivity) >= app.autoLock {
					app.lock()
				}
				app.mu.Unlock()
			}
		}
	}()
}
func (app *App) Shutdown(context.Context) { app.Lock() }
func (app *App) Touch()                   { app.mu.Lock(); defer app.mu.Unlock(); app.lastActivity = time.Now() }
func (app *App) SetAutoLockMinutes(minutes int) error {
	if minutes < 0 || minutes > 240 {
		return errors.New("invalid auto-lock interval")
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.autoLock = time.Duration(minutes) * time.Minute
	app.lastActivity = time.Now()
	return nil
}
func (app *App) lock() {
	wasOpen := app.vault != nil
	if wasOpen {
		app.vault.Lock()
	}
	app.vault = nil
	app.catalog = nil
	if app.sessionLock != nil {
		unix.Flock(int(app.sessionLock.Fd()), unix.LOCK_UN)
		app.sessionLock.Close()
		app.sessionLock = nil
	}
	if wasOpen && app.ctx != nil {
		runtime.EventsEmit(app.ctx, "vault:locked")
	}
}
func (app *App) Lock() { app.mu.Lock(); defer app.mu.Unlock(); app.lock() }
func (app *App) IsUnlocked() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.vault != nil && app.catalog != nil
}
func defaultVaultPath() string {
	if path := os.Getenv("MUT_VAULT_PATH"); filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "mut", "vault")
}
func (app *App) IsVaultConfigurated() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.vaultPath == "" {
		app.vaultPath = defaultVaultPath()
	}
	info, err := os.Stat(filepath.Join(app.vaultPath, "vault.json"))
	return err == nil && info.Mode().IsRegular()
}
func acquire(path string, create bool) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("choose an absolute library path")
	}
	if create {
		if err := storage.EnsureDir(path); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(filepath.Join(path, ".mut-lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("library is already open in another process")
	}
	return f, nil
}
func (app *App) CreateVault(path, password string) (string, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.vault != nil {
		return "", errors.New("lock the current library first")
	}
	lock, err := acquire(path, true)
	if err != nil {
		return "", err
	}
	v, key, err := vault.CreateVault(path, []byte(password))
	if err != nil {
		lock.Close()
		return "", err
	}
	c, err := catalog.Open(v)
	if err == nil {
		_, err = c.Save()
	}
	if err != nil {
		v.Lock()
		lock.Close()
		return "", err
	}
	app.vault = v
	app.catalog = c
	app.vaultPath = path
	app.sessionLock = lock
	app.lastActivity = time.Now()
	return key, nil
}
func (app *App) open(path, secret string, recovery bool) error {
	if app.vault != nil {
		return errors.New("lock the current library first")
	}
	lock, err := acquire(path, false)
	if err != nil {
		return err
	}
	var v *vault.Vault
	if recovery {
		v, err = vault.OpenVaultWithRecovery(path, secret)
	} else {
		v, err = vault.OpenVault(path, []byte(secret))
	}
	if err != nil {
		lock.Close()
		return err
	}
	c, err := catalog.Open(v)
	if err != nil {
		v.Lock()
		lock.Close()
		return err
	}
	app.vault = v
	app.catalog = c
	app.vaultPath = path
	app.sessionLock = lock
	app.lastActivity = time.Now()
	return nil
}
func (app *App) Unlock(path, password string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.open(path, password, false)
}
func (app *App) UnlockWithRecovery(path, key string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.open(path, key, true)
}
func (app *App) ListBooks() ([]catalog.Book, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return nil, vault.ErrVaultLocked
	}
	books, err := app.catalog.ListBooks()
	sort.Slice(books, func(i, j int) bool { return books[i].ID.String() < books[j].ID.String() })
	return books, err
}
func (app *App) ListShelves() ([]string, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return nil, vault.ErrVaultLocked
	}
	return app.catalog.ListShelves()
}
func (app *App) ImportBook(path string) error { return app.ImportEdition(path, false) }
func (app *App) ImportEdition(path string, keepDuplicate bool) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return vault.ErrVaultLocked
	}
	data, err := storage.ReadFile(path, storage.MaxObjectBytes)
	if err != nil {
		return err
	}
	defer clear(data)
	return importer.ImportFile(filepath.Base(path), app.vault, app.catalog, data, keepDuplicate)
}
func parseBookID(id string) (uuid.UUID, error) {
	if len(id) == 32 {
		id = id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:]
	}
	return uuid.Parse(id)
}
func (app *App) book(id string) (catalog.Book, error) {
	if app.catalog == nil {
		return catalog.Book{}, vault.ErrVaultLocked
	}
	parsed, err := parseBookID(id)
	if err != nil {
		return catalog.Book{}, errors.New("invalid book ID")
	}
	return app.catalog.GetBook(parsed)
}
func (app *App) OpenBook(id string) (readers.Document, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return readers.Document{}, err
	}
	if book.FileSize > readers.MaxBookBytes {
		return readers.Document{}, errors.New("built-in reading is limited to 64 MiB; export the original for an external viewer")
	}
	data, err := app.vault.GetObject(book.FileObjectID)
	if err != nil {
		return readers.Document{}, err
	}
	defer clear(data)
	return readers.Parse(book.Format, data)
}
func (app *App) GetCover(id string) (string, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return "", err
	}
	if book.CoverObjectID == "" {
		return "", nil
	}
	data, err := app.vault.GetObject(book.CoverObjectID)
	if err != nil {
		return "", err
	}
	defer clear(data)
	if len(data) > 8<<20 {
		return "", errors.New("cover too large")
	}
	value := string(data)
	if !strings.HasPrefix(value, "data:image/") {
		return "", errors.New("invalid cover")
	}
	return value, nil
}
func (app *App) SavePosition(id string, progress float64, locator string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return err
	}
	return app.catalog.Mutate(func() error { return app.catalog.UpdateReadingPosition(book.ID, progress, locator) })
}
func (app *App) SaveBookmarks(id string, marks []string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return err
	}
	return app.catalog.Mutate(func() error { return app.catalog.SetBookmarks(book.ID, marks) })
}
func shelfName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 240 {
		return "", errors.New("shelf name must contain 1–240 bytes")
	}
	return name, nil
}
func (app *App) CreateShelf(name string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return vault.ErrVaultLocked
	}
	name, err := shelfName(name)
	if err != nil {
		return err
	}
	return app.catalog.Mutate(func() error { return app.catalog.AddShelf(name) })
}
func (app *App) RenameShelf(oldName, name string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return vault.ErrVaultLocked
	}
	name, err := shelfName(name)
	if err != nil {
		return err
	}
	return app.catalog.Mutate(func() error { return app.catalog.RenameShelf(oldName, name) })
}
func (app *App) DeleteShelf(name string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.catalog == nil {
		return vault.ErrVaultLocked
	}
	return app.catalog.Mutate(func() error { return app.catalog.DeleteShelf(name) })
}
func (app *App) SetBookShelves(id string, shelves []string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return err
	}
	if len(shelves) > 100 {
		return errors.New("too many shelves")
	}
	for _, s := range shelves {
		if _, err := shelfName(s); err != nil {
			return err
		}
	}
	book.Shelves = append([]string{}, shelves...)
	return app.catalog.Mutate(func() error {
		for _, s := range shelves {
			found := false
			for _, existing := range app.catalog.Snapshot.Shelves {
				if s == existing {
					found = true
				}
			}
			if !found {
				if err := app.catalog.AddShelf(s); err != nil {
					return err
				}
			}
		}
		return app.catalog.AddBook(book)
	})
}
func (app *App) DeleteBook(id string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return err
	}
	return app.catalog.Mutate(func() error { return app.catalog.DeleteBook(book.ID) })
}
func (app *App) ExportOriginal(id, path string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	book, err := app.book(id)
	if err != nil {
		return err
	}
	data, err := importer.ExportOriginal(app.vault, book)
	if err != nil {
		return err
	}
	defer clear(data)
	return storage.AtomicWrite(path, data, false)
}
func (app *App) ChooseExportPath(id string) (string, error) {
	app.mu.Lock()
	book, err := app.book(id)
	app.mu.Unlock()
	if err != nil {
		return "", err
	}
	if app.ctx == nil {
		return "", errors.New("desktop dialog unavailable")
	}
	return runtime.SaveFileDialog(app.ctx, runtime.SaveDialogOptions{Title: "Экспортировать незашифрованный оригинал", DefaultFilename: filepath.Base(book.Originalname)})
}

type BackupReport struct {
	Path     string `json:"path"`
	Files    int    `json:"files"`
	Bytes    int64  `json:"bytes"`
	Verified bool   `json:"verified"`
}

func (app *App) CreateBackup(directory string) (BackupReport, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.vault == nil {
		return BackupReport{}, vault.ErrVaultLocked
	}
	destination := filepath.Join(directory, "mut-"+time.Now().UTC().Format("20060102-150405")+"-"+uuid.New().String()[:8])
	m, err := backup.CreateBackup(app.vaultPath, destination)
	if err != nil {
		return BackupReport{}, err
	}
	return BackupReport{Path: destination, Files: len(m.Files), Verified: true}, nil
}
func (app *App) VerifyBackup(path string) error { return backup.VerifyBackup(path) }
func validateVault(path, secret string, recovery bool) error {
	var v *vault.Vault
	var err error
	if recovery {
		v, err = vault.OpenVaultWithRecovery(path, secret)
	} else {
		v, err = vault.OpenVault(path, []byte(secret))
	}
	if err != nil {
		return err
	}
	defer v.Lock()
	return catalog.Verify(v)
}
func (app *App) RestoreBackup(source, target, secret string, recovery bool) (BackupReport, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.vault != nil {
		src, _ := filepath.Abs(app.vaultPath)
		dst, _ := filepath.Abs(target)
		if src == dst {
			return BackupReport{}, errors.New("cannot restore over the open library")
		}
	}
	if err := backup.RestoreValidated(source, target, func(stage string) error { return validateVault(stage, secret, recovery) }); err != nil {
		return BackupReport{}, err
	}
	return BackupReport{Path: target, Verified: true}, nil
}
func (app *App) ChangePassword(old, newPassword string) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.vault == nil {
		return vault.ErrVaultLocked
	}
	return app.vault.ChangePassword([]byte(old), []byte(newPassword))
}
