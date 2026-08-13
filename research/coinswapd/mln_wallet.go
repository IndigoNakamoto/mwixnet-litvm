package main

import (
	"fmt"

	"github.com/ltcmweb/coinswapd/mlnroute"
	"github.com/ltcmweb/ltcd/ltcutil/mweb"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
	"github.com/ltcmweb/neutrino"
)

// listWalletCoins scans MwebCoinDB unspent leaves; RewindOutput + CalculateOutputKey for keys.
func listWalletCoins(cs *neutrino.ChainService, scanKey, spendKey *mw.SecretKey) ([]*mweb.Coin, error) {
	if cs == nil || scanKey == nil {
		return nil, fmt.Errorf("chain service or scan key missing")
	}
	leafset, err := cs.MwebCoinDB.GetLeafset()
	if err != nil {
		return nil, err
	}
	if leafset == nil {
		return nil, nil
	}
	var out []*mweb.Coin
	for i := uint64(0); i < leafset.Size; i++ {
		if !leafset.Contains(i) {
			continue
		}
		utxos, err := cs.MwebCoinDB.FetchLeaves([]uint64{i})
		if err != nil || len(utxos) == 0 || utxos[0].Output == nil {
			continue
		}
		coin, err := mweb.RewindOutput(utxos[0].Output, scanKey)
		if err != nil {
			continue
		}
		coin.OutputId = utxos[0].OutputId
		if spendKey != nil {
			if sk := spendKeyForRewoundCoin(scanKey, spendKey, coin); sk != nil {
				coin.CalculateOutputKey(sk)
			}
		}
		out = append(out, coin)
	}
	return out, nil
}

func mwebBalanceTotals(coins []*mweb.Coin) (availableSat, spendableSat uint64) {
	for _, c := range coins {
		availableSat += c.Value
		if c.SpendKey != nil {
			spendableSat += c.Value
		}
	}
	return availableSat, spendableSat
}

// pickCoinExactAmount returns a coin with SpendKey set and Value == amount.
func pickCoinExactAmount(coins []*mweb.Coin, amount uint64) (*mweb.Coin, error) {
	var have []uint64
	for _, c := range coins {
		if c == nil || c.SpendKey == nil {
			continue
		}
		if c.Value == amount {
			return c, nil
		}
		have = append(have, c.Value)
	}
	return nil, mlnroute.InsufficientFunds(
		fmt.Sprintf("no spendable MWEB coin with value == %d sat (exact match required); have %v", amount, have))
}

// spendKeyForRewoundCoin returns the per-address spend secret b_i so
// CalculateOutputKey matches chain ReceiverPubKey. Master spend is never
// the output key: even index 0 is tweaked (ltcd Keychain.SpendKey).
const maxMwebAddressIndex = 4096

func spendKeyForRewoundCoin(scan, spend *mw.SecretKey, coin *mweb.Coin) *mw.SecretKey {
	if scan == nil || spend == nil || coin == nil || coin.Address == nil || coin.Address.Spend == nil {
		return nil
	}
	kc := &mweb.Keychain{Scan: scan, Spend: spend}
	want := *coin.Address.Spend
	for i := uint32(0); i < maxMwebAddressIndex; i++ {
		addr := kc.Address(i)
		if addr != nil && addr.Spend != nil && *addr.Spend == want {
			return kc.SpendKey(i)
		}
	}
	return nil
}
