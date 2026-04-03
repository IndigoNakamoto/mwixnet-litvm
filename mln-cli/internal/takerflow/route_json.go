package takerflow

import (
	"encoding/json"
	"fmt"

	"github.com/IndigoNakamoto/mwixnet-litvm/mln-cli/internal/pathfind"
)

// ParseRouteJSON decodes a route produced by pathfind / BuildRoute JSON.
func ParseRouteJSON(raw string) (*pathfind.Route, error) {
	var r pathfind.Route
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, fmt.Errorf("takerflow: route json: %w", err)
	}
	return &r, nil
}
