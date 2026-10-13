package storage

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAtomicWritePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "original")
	if err := AtomicWrite(path, []byte("original"), false); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(path, []byte("replacement"), false); err == nil {
		t.Fatal("overwrote existing file")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatal("existing data changed")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("plaintext permissions")
	}
	if err := AtomicWrite(filepath.Join(path, "child"), []byte("new"), false); err == nil {
		t.Fatal("accepted a file as parent")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("temporary files left behind")
	}
}

// Kill actual writers before and during writes. HEAD must always contain an
// entire old or new value. This checks process crashes, not loss of power.
func TestAtomicWriteProcessCrash(t *testing.T) {
	if dir := os.Getenv("MUT_STORAGE_CRASH_DIR"); dir != "" {
		data := bytes.Repeat([]byte("b"), 2<<20)
		for {
			if err := AtomicWrite(filepath.Join(dir, "HEAD"), data, true); err != nil {
				os.Exit(2)
			}
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "HEAD")
	old := bytes.Repeat([]byte("a"), 2<<20)
	next := bytes.Repeat([]byte("b"), 2<<20)
	for _, delay := range []time.Duration{time.Millisecond, 10 * time.Millisecond, 50 * time.Millisecond} {
		if err := AtomicWrite(path, old, true); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestAtomicWriteProcessCrash$")
		cmd.Env = append(os.Environ(), "MUT_STORAGE_CRASH_DIR="+dir)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(delay)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, old) && !bytes.Equal(data, next) {
			t.Fatal("partial commit after process crash")
		}
	}
}
