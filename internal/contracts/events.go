// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package contracts

import (
	"errors"
	"strings"
)

type OutputData struct {
	Stream string `json:"stream"` // stdout 或 stderr
	Text   string `json:"text"`
}

type AuditData struct {
	Target string `json:"target"` // 主机名，不包含 URL 查询串
	Tool   string `json:"tool"`
	Result string `json:"result"` // allowed、denied 或 failed
}

type Event struct {
	SessionID string           `json:"sessionId"`
	Seq       uint64           `json:"seq"`
	Type      string           `json:"type"`
	State     *SessionSnapshot `json:"state"`
	Output    *OutputData      `json:"output"`
	Audit     *AuditData       `json:"audit"`
	Error     *AppError        `json:"error"`
}

// Validate 只校验公开事件的数据形状，不执行策略、文件或网络检查。
func (e Event) Validate() error {
	if e.SessionID == "" || e.Seq == 0 {
		return errors.New("event identity is required")
	}
	payloads := 0
	if e.State != nil {
		payloads++
	}
	if e.Output != nil {
		payloads++
	}
	if e.Audit != nil {
		payloads++
	}
	if e.Error != nil {
		payloads++
	}
	if payloads != 1 {
		return errors.New("event must contain exactly one payload")
	}
	switch e.Type {
	case "state_changed":
		if e.State == nil || e.State.SessionID != e.SessionID || e.State.Seq != e.Seq {
			return errors.New("invalid state event")
		}
	case "output":
		if e.Output == nil || (e.Output.Stream != "stdout" && e.Output.Stream != "stderr") || e.Output.Text == "" || strings.ContainsRune(e.Output.Text, '\x00') {
			return errors.New("invalid output event")
		}
	case "audit":
		if e.Audit == nil || e.Audit.Target == "" || e.Audit.Tool == "" || (e.Audit.Result != "allowed" && e.Audit.Result != "denied" && e.Audit.Result != "failed") {
			return errors.New("invalid audit event")
		}
	case "error":
		if e.Error == nil || e.Error.Code == "" || e.Error.Message == "" {
			return errors.New("invalid error event")
		}
	default:
		return errors.New("unknown event type")
	}
	return nil
}
