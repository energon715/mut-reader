package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
	"mut/internal/backup"
	"mut/internal/catalog"
	"mut/internal/importer"
	"mut/internal/storage"
	"mut/internal/vault"
)

var inReader io.Reader = os.Stdin
var stdinScanner *bufio.Scanner

func getLineFromStdin() (string, error) {
	if stdinScanner == nil {
		stdinScanner = bufio.NewScanner(inReader)
	}
	if stdinScanner.Scan() {
		return strings.TrimSpace(stdinScanner.Text()), nil
	}
	if err := stdinScanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	cmd := os.Args[1]
	userArgs := os.Args[2:]

	switch cmd {
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)

	case "backup":
		err = runBackup(userArgs)

	case "verify":
		err = runVerify(userArgs)

	case "restore":
		err = runRestore(userArgs)

	case "check":
		err = runCheck(userArgs)

	case "export":
		err = runExport(userArgs)
	case "purge":
		err = runPurge(userArgs)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	usage := `vaultctl — offline management and disaster recovery tool for mut

Usage:
  vaultctl <command> [arguments]

Commands:
  backup   <vault-path> <backup-dir>           Create an encrypted backup of the vault
  verify   <backup-path>                       Verify file checksums in a backup manifest
  restore  <backup-path> <target-vault-path>   Restore a vault from an encrypted backup
  check    <vault-path>                        Check all originals and covers [--password]
  export   <vault-path> <output-dir>           Export original unencrypted book files
  purge    <vault-path>                        Delete the vault (does not guarantee secure erasure)

Options:
  -h, --help   Show this help message

Examples:
  vaultctl backup ~/.local/share/mut/vault /media/backup/mut-backup
  vaultctl verify /media/backup/mut-backup
  vaultctl restore /media/backup/mut-backup ~/.local/share/mut/restored-vault
  vaultctl check ~/.local/share/mut/vault
  vaultctl export ~/.local/share/mut/vault ~/Desktop/exported-books
  vaultctl purge ~/.local/share/mut/vault
`
	fmt.Print(usage)
}

func runBackup(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: vaultctl backup <vault-path> <backup-dir>")
	}

	vaultPath := args[0]
	backupDir := args[1]

	manifest, err := backup.CreateBackup(vaultPath, backupDir)
	if err != nil {
		return fmt.Errorf("create backup error: %w", err)
	}

	fmt.Printf("  Backup created successfully\n")
	fmt.Printf("  Files backed up:   %d\n", len(manifest.Files))
	fmt.Printf("  Catalog Snapshot:  %s\n", manifest.SnapshotID)
	fmt.Printf("  Destination:       %s\n", backupDir)

	return nil
}

func runVerify(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: vaultctl verify <backup-path>")
	}
	backupPath := args[0]
	err := backup.VerifyBackup(backupPath)
	if err != nil {
		return fmt.Errorf("verify backup error: %w", err)
	}
	fmt.Printf("  Backup verified successfully\n")

	return nil
}

func runRestore(args []string) error {
	if len(args) < 2 || len(args) > 3 || (len(args) == 3 && args[2] != "--password") {
		return fmt.Errorf("usage: vaultctl restore <backup-path> <target-vault-path> [--password]")
	}
	backupPath := args[0]
	targetPath := args[1]
	password := len(args) == 3 && args[2] == "--password"
	secret, err := readSecret(password)
	if err != nil {
		return err
	}
	err = backup.RestoreValidated(backupPath, targetPath, func(stage string) error {
		v, err := openSecret(stage, secret, password)
		if err != nil {
			return err
		}
		defer v.Lock()
		return catalog.Verify(v)
	})
	if err != nil {
		return fmt.Errorf("restore backup error: %w", err)
	}

	fmt.Printf("  Backup restored successfully\n")

	return nil
}

func runCheck(args []string) error {
	if len(args) < 1 || len(args) > 2 || (len(args) == 2 && args[1] != "--password" && args[1] != "--recovery") {
		return fmt.Errorf("usage: vaultctl check <vault-path> [--password|--recovery]")
	}
	password := len(args) == 2 && args[1] == "--password"
	secret, err := readSecret(password)
	if err != nil {
		return err
	}
	v, err := openSecret(args[0], secret, password)
	if err != nil {
		return err
	}
	defer v.Lock()
	if err := catalog.Verify(v); err != nil {
		return err
	}
	fmt.Println("  Catalog, originals and covers verified.")
	return nil
}

func runExport(args []string) error {
	if len(args) < 2 || len(args) > 3 || (len(args) == 3 && args[2] != "--password") {
		return fmt.Errorf("usage: vaultctl export <vault-path> <output-dir> [--password]")
	}
	vaultPath := args[0]
	outputDir := args[1]

	var v *vault.Vault

	password := len(args) == 3 && args[2] == "--password"
	secret, err := readSecret(password)
	if err != nil {
		return err
	}
	v, err = openSecret(vaultPath, secret, password)
	if err != nil {
		return fmt.Errorf("open vault with recovery key error: %w", err)
	}

	defer v.Lock()

	c, err := catalog.Open(v)
	if err != nil {
		return fmt.Errorf("failed to open catalog: %w", err)
	}

	books, err := c.ListBooks()
	if err != nil {
		return fmt.Errorf("failed to list books: %w", err)
	}

	if len(books) == 0 {
		fmt.Println("  Catalog is empty. Nothing to export.")
		return nil
	}

	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	fmt.Printf("  Exporting %d books to %s...\n\n", len(books), outputDir)

	exportedCount := 0
	usedNames := make(map[string]int)

	for i, book := range books {
		data, err := importer.ExportOriginal(v, book)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [%d/%d] ⚠️ Failed to export '%s': %v\n", i+1, len(books), book.Title, err)
			continue
		}

		fileName := filepath.Base(book.Originalname)
		if fileName == "" || fileName == "." || fileName == "/" {
			ext := strings.ToLower(book.Format)
			fileName = book.ID.String() + "." + filepath.Base(ext)
		}

		targetPath := filepath.Join(outputDir, fileName)
		if usedNames[fileName] > 0 {
			ext := filepath.Ext(fileName)
			base := strings.TrimSuffix(fileName, ext)
			fileName = fmt.Sprintf("%s (%d)%s", base, usedNames[fileName], ext)
			targetPath = filepath.Join(outputDir, fileName)
		}
		usedNames[filepath.Base(book.Originalname)]++

		err = storage.AtomicWrite(targetPath, data, false)
		clear(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [%d/%d] ⚠️ Failed to write '%s': %v\n", i+1, len(books), fileName, err)
			continue
		}

		fmt.Printf("  [%d/%d]  %s -> %s\n", i+1, len(books), book.Title, fileName)
		exportedCount++
	}

	fmt.Printf("\n  Export finished: %d of %d books exported successfully\n", exportedCount, len(books))
	if exportedCount != len(books) {
		return errors.New("some originals could not be exported; existing files were preserved")
	}
	return nil
}

func runPurge(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: vaultctl purge <vault-path>")
	}

	vaultPath := args[0]

	if _, err := os.Stat(filepath.Join(vaultPath, "vault.json")); err != nil {
		return fmt.Errorf("vault not found at %s: %w", vaultPath, err)
	}

	fmt.Println("  VAULT PURGE REQUEST: PUT RECOVERY KEY")
	recoveryKey, err := readRecoveryKey()
	if err != nil {
		return fmt.Errorf("read recovery key error: %w", err)
	}

	v, err := vault.OpenVaultWithRecovery(vaultPath, recoveryKey)
	if err != nil {
		return fmt.Errorf("open vault with recovery key error: %w", err)
	}
	v.Lock()
	fmt.Printf("\n⚠️  WARNING: This will PERMANENTLY DESTROY the vault at:\n    %s\n", vaultPath)
	fmt.Println("TYPE 'PURGE' TO DELETE ALL!")

	confirm, err := getLineFromStdin()
	if err != nil || strings.TrimSpace(confirm) != "PURGE" {
		fmt.Println("Purge aborted.")
		return nil
	}

	err = os.RemoveAll(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to remove vault directory: %w", err)
	}

	fmt.Println("  Vault directory deleted. Copies, snapshots and storage remnants may still exist.")
	return nil

}

func openSecret(path, secret string, password bool) (*vault.Vault, error) {
	if password {
		return vault.OpenVault(path, []byte(secret))
	}
	return vault.OpenVaultWithRecovery(path, secret)
}
func readRecoveryKey() (string, error) { return readSecret(false) }
func readSecret(password bool) (string, error) {
	label := "recovery key"
	if password {
		label = "password"
	}
	fmt.Print("Enter " + label + ": ")
	if f, ok := inReader.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		data, err := term.ReadPassword(int(f.Fd()))
		fmt.Println()
		defer clear(data)
		return string(data), err
	}
	return getLineFromStdin()
}
