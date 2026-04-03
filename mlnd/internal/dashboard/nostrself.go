package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/makerad"
	gnostr "github.com/nbd-wtf/go-nostr"
)

// NetworkPillarSelfCheck is relay visibility + binding for this operator’s replaceable ad.
type NetworkPillarSelfCheck struct {
	DTag                    string `json:"dTag"`
	RelayQueried            string `json:"relayQueried,omitempty"`
	EventFound              bool   `json:"eventFound"`
	EventID                 string `json:"eventId,omitempty"`
	CreatedAt               int64  `json:"createdAt,omitempty"`
	ContentJSON             string `json:"contentJson,omitempty"`
	SwapX25519FromRelay     string `json:"swapX25519FromRelay,omitempty"`
	LocalSwapX25519Expected string `json:"localSwapX25519Expected,omitempty"`
	SwapKeyDrift            bool   `json:"swapKeyDrift"`
	NostrKeyHashMatch       bool   `json:"nostrKeyHashMatch"`
	RegistryOK              bool   `json:"registryOk"`
	VerifyReason            string `json:"verifyReason,omitempty"`
	Error                   string `json:"error,omitempty"`
}

// FetchLatestMakerAdForDTag returns the newest matching kind-31250 event from relays, or nil if none.
func FetchLatestMakerAdForDTag(ctx context.Context, relays []string, dTag, chainID, registryHex, courtHex string) (*gnostr.Event, string, error) {
	if len(relays) == 0 {
		return nil, "", fmt.Errorf("no relays")
	}
	if strings.TrimSpace(dTag) == "" {
		return nil, "", fmt.Errorf("empty d-tag")
	}

	subCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	filter := gnostr.Filter{
		Kinds: []int{makerad.KindMakerAd},
		Tags: gnostr.TagMap{
			"t": []string{makerad.TagTMakerAd},
			"d": []string{dTag},
		},
	}

	pool := gnostr.NewSimplePool(subCtx)
	ch := pool.SubManyEose(subCtx, relays, gnostr.Filters{filter})

	var best *gnostr.Event
	var bestRelay string
	for ie := range ch {
		ev := ie.Event
		if ev == nil {
			continue
		}
		if err := makerad.VerifySignature(ev); err != nil {
			continue
		}
		parsed, err := makerad.ParseAd(ev)
		if err != nil {
			continue
		}
		if strings.TrimSpace(parsed.Content.Litvm.ChainID) != strings.TrimSpace(chainID) {
			continue
		}
		if normAddr(parsed.Content.Litvm.Registry) != normAddr(registryHex) {
			continue
		}
		if courtHex != "" && normAddr(parsed.Content.Litvm.GrievanceCourt) != normAddr(courtHex) {
			continue
		}
		if best == nil || int64(ev.CreatedAt) > int64(best.CreatedAt) {
			e := *ev
			best = &e
			if ie.Relay != nil {
				bestRelay = ie.Relay.URL
			}
		}
	}

	if best == nil {
		return nil, "", fmt.Errorf("no matching maker ad on relays (within timeout)")
	}
	return best, bestRelay, nil
}

func normAddr(a string) string {
	s := strings.TrimSpace(strings.ToLower(a))
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return s
}
