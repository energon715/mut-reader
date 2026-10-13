package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectTamperedHeaderBeforeKDF(t *testing.T) {
	for _, mutate := range []func(*Header){func(h *Header) { h.KDF.Memory = ^uint32(0) }, func(h *Header) { h.KDF.Threads = 0 }, func(h *Header) { h.KeyNonce = []byte{1} }, func(h *Header) { h.Version = 999 }} {
		path := filepath.Join(t.TempDir(), "vault")
		v, _, err := CreateVault(path, []byte("long-test-password"))
		if err != nil {
			t.Fatal(err)
		}
		h := v.header
		v.Lock()
		mutate(&h)
		data, _ := json.Marshal(h)
		os.WriteFile(filepath.Join(path, "vault.json"), data, 0600)
		if _, err := OpenVault(path, []byte("long-test-password")); err == nil {
			t.Fatal("tampered header accepted")
		}
	}
}
func TestRejectObjectAndSnapshotTraversal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault")
	v, _, err := CreateVault(path, []byte("long-test-password"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../../outside", "/tmp/outside", "xx"} {
		if _, err := v.GetObject(id); err == nil {
			t.Fatal("invalid object ID accepted")
		}
		if _, err := v.GetSnapshot(id); err == nil {
			t.Fatal("invalid snapshot ID accepted")
		}
	}
}
func FuzzHeader(f *testing.F) {
	f.Add([]byte(`{"version":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 65536 {
			t.Skip()
		}
		var h Header
		if json.Unmarshal(data, &h) == nil {
			validateHeader(h)
		}
	})
}
