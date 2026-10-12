package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mut/internal/backup"
	"mut/internal/catalog"
	"mut/internal/importer"
	"mut/internal/vault"
)

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
  check    <vault-path>                        Check vault access (password or recovery key)
  export   <vault-path> <output-dir>           Export original unencrypted book files
  purge    <vault-path>						   Delete ALL

Options:
  -h, --help   Show this help message

Examples:
  vaultctl backup ~/.local/share/mut/vault /media/backup/mut-backup
  vaultctl verify /media/backup/mut-backup
  vaultctl restore /media/backup/mut-backup ~/.local/share/mut/restored-vault
  vaultctl check ~/.local/share/mut/vault
  vaultctl export ~/.local/share/mut/vault ~/Desktop/exported-books
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
		return fmt.Errorf("  Create backup error: %w", err)
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
	bachupPAth := args[0]
	err := backup.VerifyBackup(bachupPAth)
	if err != nil {
		return fmt.Errorf("  Verify backup error: %w", err)
	}
	fmt.Printf("  Backup verified successfully\n")

	return nil
}

func runRestore(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: vaultctl restore <backup-path> <target-vault-path>")
	}
	backupPath := args[0]
	targetPath := args[1]
	err := backup.RestoreBackup(backupPath, targetPath)
	if err != nil {
		return fmt.Errorf("  Restore backup error: %w", err)
	}

	fmt.Printf("  Backup restored successfully\n")

	return nil
}

func runCheck(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vaultctl check <vault-path> [--recovery]")
	}

	vaulthPath := args[0]

	var v *vault.Vault

	recKey, err := readRecoveryKey()
	if err != nil {
		return fmt.Errorf("read recovery error: %w", err)
	}
	v, err = vault.OpenVaultWithRecovery(vaulthPath, recKey)
	if err != nil {
		return fmt.Errorf("open vault with recovery error: %w", err)
	}

	headID, _ := v.GetHead()

	fmt.Println("  Access granted! Vault is valid.")
	if headID != "" {
		fmt.Printf("  Current Snapshot: %s\n", headID)
	}
	return nil

}

func runExport(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: vaultctl export <vault-path> <output-dir>")
	}
	vaultPath := args[0]
	outputDir := args[1]

	var v *vault.Vault

	recoveryKey, err := readRecoveryKey()
	if err != nil {
		return fmt.Errorf("read recoveryKey error: %w", err)
	}
	v, err = vault.OpenVaultWithRecovery(vaultPath, recoveryKey)
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
			fileName = fmt.Sprintf("%s.%s", book.Title, ext)
		}

		targetPath := filepath.Join(outputDir, fileName)
		if usedNames[fileName] > 0 {
			ext := filepath.Ext(fileName)
			base := strings.TrimSuffix(fileName, ext)
			fileName = fmt.Sprintf("%s (%d)%s", base, usedNames[fileName], ext)
			targetPath = filepath.Join(outputDir, fileName)
		}
		usedNames[filepath.Base(book.Originalname)]++

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  [%d/%d] ⚠️ Failed to write '%s': %v\n", i+1, len(books), fileName, err)
			continue
		}

		fmt.Printf("  [%d/%d]  %s -> %s\n", i+1, len(books), book.Title, fileName)
		exportedCount++
	}

	fmt.Printf("\n  Export finished: %d of %d books exported successfully\n", exportedCount, len(books))
	return nil
}

func runPurge(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: vaultctl restore <vault-path> ")
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

	reader := bufio.NewReader(os.Stdin)
	confirm, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(confirm) != "PURGE" {
		fmt.Println("Purge aborted.")
		return nil
	}

	fmt.Println("  Cryptographically shredding vault headers...")
	err = shredFile(filepath.Join(vaultPath, "vault.json"))
	if err != nil {
		return fmt.Errorf("shred file error: %w", err)
	}
	err = shredFile(filepath.Join(vaultPath, "HEAD"))
	if err != nil {
		return fmt.Errorf("shred file error: %w", err)
	}

	_ = filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			err = shredFile(path)
			if err != nil {
				return err
			}
		}
		return nil
	})

	err = os.RemoveAll(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to remove vault directory: %w", err)
	}

	fmt.Println("  Vault has been permanently and irreversibly destroyed.")
	return nil

}

func readRecoveryKey() (string, error) {
	fmt.Print("Enter recovery key: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read recovery key: %w", err)
	}
	return strings.TrimSpace(input), nil

}

func shredFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	randomData := make([]byte, info.Size())
	_, _ = rand.Read(randomData)

	_, err = file.Write(randomData)
	if err != nil {
		return err
	}

	_ = file.Sync()
	return nil
}
