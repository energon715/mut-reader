package vault

import (
	"fmt"
	"uuid"
)

type Header struct {
	Version          int       `json:"version"`
	VaultID          uuid.UUID `json:"vault_id"`
	KDF              KDF       `json:"kdf"`
	WrappedMasterKey []byte    `json:"wrapped_master_key"`
	KeyNonce         []byte    `json:"key_nonce"`
	Recovery         []byte    `json:"mobile_recovery"`
}

type KDF struct {
	Salt    []byte `json:"salt"`
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
}

func validateHeader(h Header) error {
	if h.Version != 1 || h.VaultID == uuid.Nil() {
		return fmt.Errorf("unsupported or invalid vault header")
	}
	if len(h.KeyNonce) != 24 || len(h.WrappedMasterKey) != 48 || len(h.Recovery) != 72 {
		return fmt.Errorf("invalid key envelope")
	}
	if len(h.KDF.Salt) < 16 || len(h.KDF.Salt) > 64 || h.KDF.Time < 1 || h.KDF.Time > 10 || h.KDF.Memory < 8*1024 || h.KDF.Memory > 256*1024 || h.KDF.Threads < 1 || h.KDF.Threads > 16 {
		return fmt.Errorf("invalid KDF parameters")
	}
	return nil
}
