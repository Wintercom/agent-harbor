package desktop

import (
	"errors"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"testing"
)

func TestWorkspaceDistinguishesCancelAndFailure(t *testing.T) {
	canceled := workspaceResult("", nil)
	if !canceled.Cancelled || canceled.Error != nil {
		t.Fatal(canceled)
	}
	failed := workspaceResult("", errors.New("不可回显的内部错误"))
	if failed.Cancelled || failed.Error == nil || failed.Error.Code != contracts.ErrInternal || failed.Error.Message == "不可回显的内部错误" {
		t.Fatal(failed)
	}
	selected := workspaceResult("/demo/project", nil)
	if selected.Path != "/demo/project" || selected.Error != nil || selected.Cancelled {
		t.Fatal(selected)
	}
}
