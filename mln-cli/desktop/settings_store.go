//go:build wails

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/config"
)

const settingsFile = "settings.json"

type settingsStore struct {
	mu   sync.Mutex
	path string
}

func newSettingsStore() (*settingsStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	appDir := filepath.Join(dir, "mln-wallet")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir settings: %w", err)
	}
	return &settingsStore{path: filepath.Join(appDir, settingsFile)}, nil
}

func (s *settingsStore) load() (config.NetworkSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out config.NetworkSettings
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultNetworkSettings(), nil
		}
		return out, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return defaultNetworkSettings(), nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("settings json: %w", err)
	}
	return out, nil
}

func (s *settingsStore) save(net config.NetworkSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func defaultNetworkSettings() config.NetworkSettings {
	return config.NetworkSettings{
		NostrRelays:       append([]string(nil), config.DefaultNostrRelays...),
		LitvmChainID:      config.DefaultLitvmChainID,
		LitvmHTTPURL:      config.DefaultLitvmHTTPURL,
		DefaultSidecarURL: "http://127.0.0.1:8080/v1/swap",
		ScoutTimeout:      "30s",
		ForgerHTTPTimeout: "10s",
	}
}
