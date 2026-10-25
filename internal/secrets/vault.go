package secrets

// vault 保存密钥本体；元信息在 Store.byID，不含明文 Key。
type vault interface {
	Set(id string, secret []byte) error
	Get(id string) ([]byte, error)
	Delete(id string) error
}

type memoryVault struct {
	m map[string][]byte
}

func newMemoryVault() *memoryVault {
	return &memoryVault{m: make(map[string][]byte)}
}

func (v *memoryVault) Set(id string, secret []byte) error {
	v.m[id] = append([]byte(nil), secret...)
	return nil
}

func (v *memoryVault) Get(id string) ([]byte, error) {
	b, ok := v.m[id]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), b...), nil
}

func (v *memoryVault) Delete(id string) error {
	if _, ok := v.m[id]; !ok {
		return ErrNotFound
	}
	delete(v.m, id)
	return nil
}
