package main

import (
	"encoding/json"
	"os"
)

// loadPubkeyMapJSON loads tor URL → 64-hex pubkey map (Approach C).
func loadPubkeyMapJSON(path string) (map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
