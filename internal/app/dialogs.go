package app

import (
	"errors"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
	"os"
	"path/filepath"
)

type Paths struct {
	Library string `json:"library"`
	Backup  string `json:"backup"`
	Restore string `json:"restore"`
	Imports string `json:"imports"`
}

func (app *App) GetDefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{defaultVaultPath(), filepath.Join(home, "Documents", "mut-backups"), filepath.Join(home, "Documents", "mut-restored"), filepath.Join(home, "Downloads")}, nil
}
func dialogDirectory(path string) string {
	if !filepath.IsAbs(path) {
		path, _ = os.UserHomeDir()
	}
	for path != "" {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return ""
}
func (app *App) ChooseDirectory(title, path string) (string, error) {
	if app.ctx == nil {
		return "", errors.New("desktop dialog unavailable")
	}
	return runtime.OpenDirectoryDialog(app.ctx, runtime.OpenDialogOptions{Title: title, DefaultDirectory: dialogDirectory(path), CanCreateDirectories: true, ShowHiddenFiles: true})
}
func (app *App) ChooseFiles(path string) ([]string, error) {
	if app.ctx == nil {
		return nil, errors.New("desktop dialog unavailable")
	}
	return runtime.OpenMultipleFilesDialog(app.ctx, runtime.OpenDialogOptions{Title: "Добавить книги", DefaultDirectory: dialogDirectory(path)})
}
func (app *App) ListImportFiles(directory string) ([]string, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var result []string
	err = fs.WalkDir(root.FS(), ".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if len(result) >= 10000 {
			return errors.New("folder import limit: 10000 files")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			result = append(result, filepath.Join(directory, filepath.FromSlash(path)))
		}
		return nil
	})
	return result, err
}
