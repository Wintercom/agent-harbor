package dsh

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/liyufei916/agent-harbor/internal/runtime"
)

//go:embed conversation.mjs
var conversationDriver string

// NewConversation 使用固定受信驱动保留同一 Agent 的多轮上下文。
func NewConversation(node, installation, model string) (*Kit, error) {
	kit, err := New(node, installation, model)
	if err != nil {
		return nil, err
	}
	kit.conversation = true
	return kit, nil
}

func conversationFile(root string) runtime.ConfigFile {
	source := conversationDriver
	for marker, pkg := range map[string]string{"__AGENT_MODULE__": "dsh-agent", "__LLM_MODULE__": "dsh-llm", "__SESSION_MODULE__": "dsh-session"} {
		path := filepath.Join(root, "node_modules", "@deepseek-ai", pkg, "lib", "index.js")
		encoded, _ := json.Marshal((&url.URL{Scheme: "file", Path: path}).String())
		source = strings.ReplaceAll(source, marker, string(encoded))
	}
	return runtime.ConfigFile{RelativePath: "harbor-conversation.mjs", Content: []byte(source)}
}

func encodeConversationMessage(data []byte) ([]byte, error) {
	if !utf8.Valid(data) || len(data) == 0 || len(data) > 32768 || strings.TrimSpace(string(data)) == "" || strings.ContainsRune(string(data), 0) {
		return nil, kitError("INVALID_ARGUMENT")
	}
	encoded, err := json.Marshal(map[string]string{"text": string(data)})
	if err != nil {
		return nil, kitError("INVALID_ARGUMENT")
	}
	return append(encoded, '\n'), nil
}
