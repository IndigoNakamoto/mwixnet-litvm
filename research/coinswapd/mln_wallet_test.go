package main

import (
	"crypto/rand"
	"testing"

	"github.com/ltcmweb/ltcd/ltcutil/mweb"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
)

func TestSpendKeyForRewoundCoin_matchesReceiverPubKey(t *testing.T) {
	t.Parallel()
	var scan, spend mw.SecretKey
	for {
		if _, err := rand.Read(scan[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := rand.Read(spend[:]); err != nil {
			t.Fatal(err)
		}
		if secretScalarOK(&scan) && secretScalarOK(&spend) {
			break
		}
	}
	kc := &mweb.Keychain{Scan: &scan, Spend: &spend}

	for _, idx := range []uint32{0, 7, 100} {
		idx := idx
		t.Run(itoa(idx), func(t *testing.T) {
			t.Parallel()
			addr := kc.Address(idx)
			var skSend mw.SecretKey
			if _, err := rand.Read(skSend[:]); err != nil {
				t.Fatal(err)
			}
			out, _, _ := mweb.CreateOutput(&mweb.Recipient{Value: 10_000_000, Address: addr}, &skSend)
			coin, err := mweb.RewindOutput(out, &scan)
			if err != nil {
				t.Fatalf("rewind: %v", err)
			}
			sk := spendKeyForRewoundCoin(&scan, &spend, coin)
			if sk == nil {
				t.Fatal("index not found")
			}
			coin.CalculateOutputKey(sk)
			if coin.SpendKey == nil {
				t.Fatal("SpendKey unset")
			}
			if *coin.SpendKey.PubKey() != out.ReceiverPubKey {
				t.Fatal("SpendKey.PubKey != chain ReceiverPubKey (would fail validateOnion)")
			}
			if *sk == spend {
				t.Fatal("must use tweaked SpendKey(index), not master spend")
			}
		})
	}
}

func itoa(u uint32) string {
	if u == 0 {
		return "0"
	}
	var b [10]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}
