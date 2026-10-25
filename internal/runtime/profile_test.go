package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOnlyInstallationHasNoRecursiveExecution(t *testing.T) {
	root := t.TempDir()
	install := filepath.Join(root, "installation")
	if err := os.Mkdir(install, 0700); err != nil {
		t.Fatal(err)
	}
	profile, err := makeProfile(root, root, []string{install}, nil)
	if err != nil {
		t.Fatal(err)
	}
	quoted, _ := sbQuote(install)
	if !strings.Contains(profile, "(allow file-read* file-map-executable (subpath "+quoted+"))") {
		t.Fatal("缺少安装目录只读授权")
	}
	if strings.Contains(profile, "(allow process-exec (subpath") || strings.Contains(profile, "(allow process-exec (literal "+quoted) {
		t.Fatal("目录不应授予执行权")
	}
}

func TestDSHEnvironmentCannotEscapePrivateHome(t *testing.T) {
	if _, err := launchEnvironment("/private/home", map[string]string{"DSH_HOME": "/Users/host/.dsh"}); err == nil {
		t.Fatal("允许借用宿主 HOME")
	}
	if _, err := launchEnvironment("/private/home", map[string]string{"DSH_TELEMETRY_DISABLED": ""}); err == nil {
		t.Fatal("允许关闭遥测禁用约束")
	}
}
