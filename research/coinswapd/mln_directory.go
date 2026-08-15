package main

import (
	"crypto/ecdh"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ltcmweb/coinswapd/config"
	"github.com/ltcmweb/coinswapd/mlnroute"
)

const mlnDirectoryHops = 3

type mlnDirectoryHop struct {
	Tor              string `json:"tor"`
	SwapX25519PubHex string `json:"swapX25519PubHex"`
	FeeMinSat        uint64 `json:"feeMinSat"`
}

type mlnDirectoryFile struct {
	AmountSat uint64            `json:"amountSat"`
	Hops      []mlnDirectoryHop `json:"hops"`
}

func loadMLNDirectoryPeers(path string) ([]config.Node, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mln-directory: read %s: %w", path, err)
	}
	var d mlnDirectoryFile
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("mln-directory: json: %w", err)
	}
	if len(d.Hops) != mlnDirectoryHops {
		return nil, fmt.Errorf("mln-directory: need exactly %d hops, got %d", mlnDirectoryHops, len(d.Hops))
	}
	if d.AmountSat == 0 {
		return nil, fmt.Errorf("mln-directory: amountSat must be positive")
	}
	var feeSum uint64
	nodes := make([]config.Node, mlnDirectoryHops)
	for i, h := range d.Hops {
		tor := strings.TrimSpace(h.Tor)
		if tor == "" {
			return nil, fmt.Errorf("mln-directory: hop %d: tor is required", i)
		}
		k := strings.TrimSpace(h.SwapX25519PubHex)
		if len(k) != 64 {
			return nil, fmt.Errorf("mln-directory: hop %d: swapX25519PubHex must be 64 hex digits", i)
		}
		for _, c := range k {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return nil, fmt.Errorf("mln-directory: hop %d: swapX25519PubHex must be lowercase hex", i)
			}
		}
		if _, err := hex.DecodeString(k); err != nil {
			return nil, fmt.Errorf("mln-directory: hop %d: swapX25519PubHex: %w", i, err)
		}
		if h.FeeMinSat == 0 {
			return nil, fmt.Errorf("mln-directory: hop %d: feeMinSat must be positive", i)
		}
		next := feeSum + h.FeeMinSat
		if next < feeSum {
			return nil, fmt.Errorf("mln-directory: hop %d: fee overflow", i)
		}
		feeSum = next
		nodes[i] = config.NewNode(tor, k)
	}
	if feeSum >= d.AmountSat {
		return nil, fmt.Errorf("mln-directory: sum of feeMinSat (%d) must be less than amountSat (%d)", feeSum, d.AmountSat)
	}
	if len(nodes) != mlnroute.ExpectedHops {
		return nil, fmt.Errorf("mln-directory: internal hop count")
	}
	return nodes, nil
}

func applyMLNDirectory(ss *swapService, path string, hopIndex int, localTaker bool, pub *ecdh.PublicKey) error {
	if hopIndex < 0 || hopIndex >= mlnDirectoryHops {
		return fmt.Errorf("mln-hop-index must be 0..%d when -mln-directory is set", mlnDirectoryHops-1)
	}
	if localTaker && hopIndex != 0 {
		return fmt.Errorf("-mln-local-taker requires -mln-hop-index 0")
	}
	peers, err := loadMLNDirectoryPeers(path)
	if err != nil {
		return err
	}
	want := peers[hopIndex].PubKey()
	if pub == nil || want == nil || !want.Equal(pub) {
		return fmt.Errorf("mln-directory: hop %d swapX25519PubHex does not match this process -k public key", hopIndex)
	}
	ss.nodes = peers
	ss.mlnPeers = peers
	ss.nodeIndex = hopIndex
	fmt.Println("mln-directory: hop", hopIndex+1, "of", len(peers), "(public mesh probe skipped)")
	return nil
}

// pinMLNDirectoryForTaker loads hop URLs/keys for a Proof B taker client.
// This process is not a mixnode: -k is not checked against the directory.
func pinMLNDirectoryForTaker(ss *swapService, path string) error {
	peers, err := loadMLNDirectoryPeers(path)
	if err != nil {
		return err
	}
	ss.nodes = peers
	ss.mlnPeers = peers
	ss.nodeIndex = 0
	ss.submitRemote = true
	fmt.Println("mln-submit-remote: taker client; hop 0", peers[0].Url, "(not a mixnode)")
	return nil
}
