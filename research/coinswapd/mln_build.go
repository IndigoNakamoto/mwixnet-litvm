package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/ltcmweb/coinswapd/mlnroute"
	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcmweb/ltcd/chaincfg"
	"github.com/ltcmweb/ltcd/ltcutil"
	"github.com/ltcmweb/ltcd/ltcutil/mweb"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
)

// buildOnionFromMLNRoute builds onion.Onion from MLN JSON + resolved X25519 pubkeys + wallet coin.
// Kernel/stealth blinds are random; MW value conservation across peels must hold for swap to
// succeed at peel time — operators may need an external solver or future blind derivation here.
func buildOnionFromMLNRoute(
	req *mlnroute.Request,
	peerPub []*ecdh.PublicKey,
	coin *mweb.Coin,
) (*onion.Onion, error) {
	if req == nil || len(peerPub) != mlnroute.ExpectedHops || coin == nil || coin.SpendKey == nil {
		return nil, mlnroute.InvalidParams("internal: bad build args")
	}

	feeSum := mlnroute.FeeSum(req)
	if feeSum > req.Amount {
		return nil, mlnroute.InvalidParams("fees exceed amount")
	}
	outVal := req.Amount - feeSum
	if outVal == 0 {
		return nil, mlnroute.InvalidParams("output value after fees is zero")
	}

	addr, err := ltcutil.DecodeAddress(strings.TrimSpace(req.Destination), &chaincfg.MainNetParams)
	if err != nil {
		return nil, mlnroute.InvalidParams(fmt.Sprintf("destination: %v", err))
	}
	mwAddr, ok := addr.(*ltcutil.AddressMweb)
	if !ok {
		return nil, mlnroute.InvalidParams("destination must be MWEB address")
	}
	stealth := mwAddr.StealthAddress()

	var ephemeralKey mw.SecretKey
	if _, err := rand.Read(ephemeralKey[:]); err != nil {
		return nil, mlnroute.Internal(err.Error())
	}

	lastOut, blind, _ := mweb.CreateOutput(&mweb.Recipient{
		Value:   outVal,
		Address: stealth,
	}, &ephemeralKey)
	mweb.SignOutput(lastOut, outVal, blind, &ephemeralKey)

	var inputKey mw.SecretKey
	if _, err := rand.Read(inputKey[:]); err != nil {
		return nil, mlnroute.Internal(err.Error())
	}
	input := mweb.CreateInput(coin, &inputKey)

	hops := make([]*onion.Hop, mlnroute.ExpectedHops)
	for i := 0; i < mlnroute.ExpectedHops; i++ {
		var kb, sb mw.BlindingFactor
		if _, err := rand.Read(kb[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		if _, err := rand.Read(sb[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		hops[i] = &onion.Hop{
			PubKey:       peerPub[i],
			KernelBlind:  kb,
			StealthBlind: sb,
			Fee:          req.Route[i].FeeMinSat,
		}
	}
	hops[len(hops)-1].Output = lastOut

	o, err := onion.New(hops)
	if err != nil {
		return nil, mlnroute.OnionOrCrypto(fmt.Sprintf("onion.New: %v", err))
	}
	o.Sign(input, coin.SpendKey)

	if !o.VerifySig() {
		return nil, mlnroute.OnionOrCrypto("onion owner signature verify failed")
	}
	return o, nil
}
