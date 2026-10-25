//go:build darwin

package secrets

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const keychainService = "com.liyufei916.agent-harbor"

type keychainVault struct{}

func newPlatformVault(dir string) vault {
	v := keychainVault{}
	probeID := "__harbor_vault_probe__"
	if err := v.Set(probeID, []byte("ok")); err != nil {
		return newFileVault(dir)
	}
	_ = v.Delete(probeID)
	return v
}

func (keychainVault) Set(id string, secret []byte) error {
	_ = keychainVault{}.Delete(id)
	cmd := exec.Command("/usr/bin/security", "add-generic-password",
		"-s", keychainService,
		"-a", id,
		"-w", string(secret),
		"-U",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain set: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (keychainVault) Get(id string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password",
		"-s", keychainService,
		"-a", id,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, ErrNotFound
	}
	return bytes.TrimSpace(out), nil
}

func (keychainVault) Delete(id string) error {
	cmd := exec.Command("/usr/bin/security", "delete-generic-password",
		"-s", keychainService,
		"-a", id,
	)
	_ = cmd.Run() // 不存在也当成功
	return nil
}
