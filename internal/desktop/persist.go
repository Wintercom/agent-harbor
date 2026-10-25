package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

type persistedState struct {
	Prefs contracts.UIPrefs          `json:"prefs"`
	Hosts []contracts.RememberedHost `json:"hosts"`
}

func (a *App) statePath() string {
	if a.dataDir == "" {
		return ""
	}
	return filepath.Join(a.dataDir, "ui-state.json")
}

func (a *App) loadPersistedState() {
	path := a.statePath()
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var st persistedState
	if json.Unmarshal(raw, &st) != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prefs = st.Prefs
	if a.hosts == nil {
		a.hosts = make(map[string]contracts.RememberedHost)
	}
	for _, h := range st.Hosts {
		if h.Host == "" {
			continue
		}
		a.hosts[hostMapKey(h.Workspace, h.Host)] = h
	}
}

func (a *App) savePersistedState() {
	path := a.statePath()
	if path == "" {
		return
	}
	a.mu.Lock()
	st := persistedState{Prefs: a.prefs, Hosts: make([]contracts.RememberedHost, 0, len(a.hosts))}
	for _, h := range a.hosts {
		st.Hosts = append(st.Hosts, h)
	}
	a.mu.Unlock()
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, raw, 0600)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	// macOS: ~/Library/Application Support/AgentHarbor
	return filepath.Join(home, "Library", "Application Support", "AgentHarbor")
}
