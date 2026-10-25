package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

type metaFile struct {
	Accounts []contracts.AccountInfo `json:"accounts"`
}

// NewPersistentStore 元信息落盘；密钥走平台 vault（Darwin=Keychain）。
func NewPersistentStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{
		byID:     make(map[string]*record),
		vault:    newPlatformVault(filepath.Join(dir, "keys")),
		metaPath: filepath.Join(dir, "accounts.json"),
	}
	if err := s.loadMeta(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadMeta() error {
	raw, err := os.ReadFile(s.metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var mf metaFile
	if err := json.Unmarshal(raw, &mf); err != nil {
		return err
	}
	for _, meta := range mf.Accounts {
		if meta.ID == "" {
			continue
		}
		s.byID[meta.ID] = &record{meta: meta}
	}
	return nil
}

func (s *Store) saveMetaLocked() error {
	if s.metaPath == "" {
		return nil
	}
	mf := metaFile{Accounts: make([]contracts.AccountInfo, 0, len(s.byID))}
	for _, rec := range s.byID {
		mf.Accounts = append(mf.Accounts, rec.meta)
	}
	raw, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, raw, 0600)
}
