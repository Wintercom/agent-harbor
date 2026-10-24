package gate

import "bytes"

const redactedText = "[redacted]"

// secretRedactor 在分块流上抹掉已知密钥，保留可能跨块的前缀。
type secretRedactor struct {
	secrets [][]byte
	keep    int
	held    []byte
}

func newSecretRedactor(secrets []string) *secretRedactor {
	r := &secretRedactor{}
	for _, s := range secrets {
		if len(s) < 8 {
			continue
		}
		b := []byte(s)
		r.secrets = append(r.secrets, b)
		if n := len(b) - 1; n > r.keep {
			r.keep = n
		}
	}
	return r
}

func (r *secretRedactor) scrub(in []byte) []byte {
	out := in
	mark := []byte(redactedText)
	for _, s := range r.secrets {
		out = bytes.ReplaceAll(out, s, mark)
	}
	return out
}

// Push 返回可以安全写出的前缀。
func (r *secretRedactor) Push(in []byte) []byte {
	if len(r.secrets) == 0 {
		return in
	}
	data := append(r.held, in...)
	r.held = nil
	data = r.scrub(data)
	if r.keep <= 0 || len(data) <= r.keep {
		r.held = append([]byte(nil), data...)
		return nil
	}
	r.held = append([]byte(nil), data[len(data)-r.keep:]...)
	return append([]byte(nil), data[:len(data)-r.keep]...)
}

// Flush 写出剩余缓冲。调用方应在流结束时调用。
func (r *secretRedactor) Flush() []byte {
	if len(r.held) == 0 {
		return nil
	}
	out := r.scrub(r.held)
	r.held = nil
	return out
}
