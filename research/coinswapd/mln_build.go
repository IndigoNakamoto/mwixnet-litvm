package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/ltcmweb/coinswapd/mlnroute"
	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcmweb/ltcd/chaincfg"
	"github.com/ltcmweb/ltcd/ltcutil"
	"github.com/ltcmweb/ltcd/ltcutil/mweb"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
)

// buildOnionFromMLNRoute builds onion.Onion from MLN JSON + resolved X25519 pubkeys + wallet coin.
// Hop kernel blinds k0,k1 are random; k2 is derived so peeled commitments match CreateOutput's
// Pedersen blind after fees. Hop stealth blinds s0,s1 are random; s2 closes the sender key
// so peeled stealth sums match wire.MwebOutput.SenderPubKey (see mln_peel.go).
func buildOnionFromMLNRoute(
	req *mlnroute.Request,
	peerPub []*ecdh.PublicKey,
	coin *mweb.Coin,
) (*onion.Onion, error) {
	if req == nil || len(peerPub) != mlnroute.ExpectedHops || coin == nil ||
		coin.SpendKey == nil || coin.Blind == nil {
		return nil, mlnroute.InvalidParams("internal: bad build args")
	}

	if coin.Value != req.Amount {
		return nil, mlnroute.InvalidParams("coin value must equal route amount")
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

	bIn := mw.BlindSwitch(coin.Blind, coin.Value)

	const maxOuter = 128
	const maxKernel = 64
	for outer := 0; outer < maxOuter; outer++ {
		var skInput, skSend, s0, s1 mw.SecretKey
		if _, err := rand.Read(skInput[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		if _, err := rand.Read(skSend[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		if _, err := rand.Read(s0[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}
		if _, err := rand.Read(s1[:]); err != nil {
			return nil, mlnroute.Internal(err.Error())
		}

		if !secretScalarOK(&skInput) || !secretScalarOK(&skSend) ||
			!secretScalarOK(&s0) || !secretScalarOK(&s1) {
			continue
		}

		tEff := coin.SpendKey.Sub(&skInput)
		s2Ptr := skSend.Sub(tEff).Sub(&s0).Sub(&s1)
		if !secretScalarOK(s2Ptr) {
			continue
		}

		lastOut, maskBlind, _ := mweb.CreateOutput(&mweb.Recipient{
			Value:   outVal,
			Address: stealth,
		}, &skSend)
		mweb.SignOutput(lastOut, outVal, maskBlind, &skSend)

		bSw := mw.BlindSwitch(maskBlind, outVal)

		var k0, k1 mw.BlindingFactor
		for kTry := 0; kTry < maxKernel; kTry++ {
			if _, err := rand.Read(k0[:]); err != nil {
				return nil, mlnroute.Internal(err.Error())
			}
			if _, err := rand.Read(k1[:]); err != nil {
				return nil, mlnroute.Internal(err.Error())
			}
			k2 := *bSw.Sub(bIn).Sub(&k0).Sub(&k1)
			if !kernelScalarOK(&k0) || !kernelScalarOK(&k1) || !kernelScalarOK(&k2) {
				continue
			}

			hops := []*onion.Hop{
				{
					PubKey:       peerPub[0],
					KernelBlind:  k0,
					StealthBlind: stealthBlindFromSecret(&s0),
					Fee:          req.Route[0].FeeMinSat,
				},
				{
					PubKey:       peerPub[1],
					KernelBlind:  k1,
					StealthBlind: stealthBlindFromSecret(&s1),
					Fee:          req.Route[1].FeeMinSat,
				},
				{
					PubKey:       peerPub[2],
					KernelBlind:  k2,
					StealthBlind: stealthBlindFromSecret(s2Ptr),
					Fee:          req.Route[2].FeeMinSat,
					Output:       lastOut,
				},
			}

			o, err := onion.New(hops)
			if err != nil {
				return nil, mlnroute.OnionOrCrypto(fmt.Sprintf("onion.New: %v", err))
			}
			input := mweb.CreateInput(coin, &skInput)
			o.Sign(input, coin.SpendKey)

			if !o.VerifySig() {
				return nil, mlnroute.OnionOrCrypto("onion owner signature verify failed")
			}
			return o, nil
		}
	}

	return nil, mlnroute.OnionOrCrypto("MW onion solver exhausted retries (kernel/stealth scalar bounds)")
}

func kernelScalarOK(b *mw.BlindingFactor) bool {
	var k secp256k1.ModNScalar
	return k.SetBytes((*[32]byte)(b)) == 0
}

func secretScalarOK(s *mw.SecretKey) bool {
	var k secp256k1.ModNScalar
	return k.SetBytes((*[32]byte)(s)) == 0
}

func stealthBlindFromSecret(s *mw.SecretKey) mw.BlindingFactor {
	var b mw.BlindingFactor
	copy(b[:], s[:])
	return b
}
