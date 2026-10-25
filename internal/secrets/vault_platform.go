//go:build !darwin

package secrets

func newPlatformVault(dir string) vault {
	return newFileVault(dir)
}
