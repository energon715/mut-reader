package main

import (
	"fmt"
	"os"

	"mut/internal/backup"
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
		return fmt.Errorf("create backup error: %w", err)
	}

	fmt.Printf("  Backup created successfully\n")
	fmt.Printf("  Files backed up:   %d\n", len(manifest.Files))
	fmt.Printf("  Catalog Snapshot:  %s\n", manifest.SnapshotID)
	fmt.Printf("  Destination:       %s\n", backupDir)

	return nil
}

func runVerify(args []string) error {
	return fmt.Errorf("verify command is not implemented yet")
}

func runRestore(args []string) error {
	return fmt.Errorf("restore command is not implemented yet")
}

func runCheck(args []string) error {
	return fmt.Errorf("check command is not implemented yet")
}

func runExport(args []string) error {
	return fmt.Errorf("export command is not implemented yet")
}

func runPurge(args []string) error {

}
