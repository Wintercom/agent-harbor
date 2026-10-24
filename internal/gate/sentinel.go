package gate

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

const sentinelPrefix = "harbor_sentinel_"

// newSentinel 生成至少 32 字节安全随机会话能力令牌。
func newSentinel() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return sentinelPrefix + hex.EncodeToString(b[:]), nil
}

// parseBearerSentinel 从 Authorization 头提取 sentinel。
func parseBearerSentinel(value string) (token string, ok bool) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		value = strings.TrimSpace(value[7:])
	}
	if !strings.HasPrefix(value, sentinelPrefix) {
		return "", false
	}
	if len(value) < len(sentinelPrefix)+64 { // 32 bytes hex
		return "", false
	}
	return value, true
}

// looksLikeSentinel 判断值是否为本系统 sentinel。
func looksLikeSentinel(value string) bool {
	_, ok := parseBearerSentinel(value)
	return ok
}
