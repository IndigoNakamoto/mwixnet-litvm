package main

import (
	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
	"github.com/ltcmweb/ltcd/wire"
)

// Peel math (must match swap.go peelOnions):
//
// Starting from the taker input commitment C0 = SwitchCommit(coin.Blind, coin.Value) and
// stealth base S0 = input.OutputPubKey.Sub(input.InputPubKey), each hop i applies:
//
//	C{i+1} = C{i} + NewCommitment(KernelBlind_i, 0) - NewCommitment(0, Fee_i)
//	S{i+1} = S{i} + PubKey(StealthBlind_i)
//
// After three hops with fees f0,f1,f2 and coin value V = amount, the H-weight is
// V - f0 - f1 - f2 = outVal (requires V = amount and outVal = amount - sum(fees))).
//
// The G-blinding coefficients satisfy:
//
//	BlindSwitch(maskBlind, outVal) = BlindSwitch(coin.Blind, V) + k0 + k1 + k2
//
// when the last hop's output was built with CreateOutput/SignOutput using that maskBlind,
// and KernelBlind_i = k_i.
//
// Stealth (secp256k1): with tEff = spendSecret - inputSecret (scalars mod n),
// S0 = Pub(tEff). Choosing senderSecret and hop stealth scalars s0,s1,s2 with
// s0+s1+s2 = senderSecret - tEff gives S0 + Pub(s0)+Pub(s1)+Pub(s2) = Pub(senderSecret),
// matching wire.MwebOutput.SenderPubKey from CreateOutput.

// peelCommitStep applies one peel's commitment update (same as swap.go peelOnions).
func peelCommitStep(commit *mw.Commitment, hop *onion.Hop) *mw.Commitment {
	return commit.Add(mw.NewCommitment(&hop.KernelBlind, 0)).
		Sub(mw.NewCommitment(&mw.BlindingFactor{}, hop.Fee))
}

// peelStealthStep applies one peel's stealth sum update.
func peelStealthStep(stealthSum *mw.PublicKey, hop *onion.Hop) *mw.PublicKey {
	sb := mw.SecretKey(hop.StealthBlind)
	return stealthSum.Add(sb.PubKey())
}

// inputStealthBase returns input.OutputPubKey.Sub(input.InputPubKey) for CreateInput(coin, inputKey).
func inputStealthBase(input *wire.MwebInput) *mw.PublicKey {
	return input.OutputPubKey.Sub(input.InputPubKey)
}
