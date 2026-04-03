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
			coin.CalculateOutputKey(spendKey)
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
	for _, c := range coins {
		if c == nil || c.SpendKey == nil {
			continue
		}
		if c.Value == amount {
			return c, nil
		}
	}
	return nil, mlnroute.InsufficientFunds(
		fmt.Sprintf("no spendable MWEB coin with value == %d sat (exact match required)", amount))
}
