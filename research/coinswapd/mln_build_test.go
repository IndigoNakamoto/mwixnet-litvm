package main

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/ltcmweb/coinswapd/mlnroute"
	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcmweb/ltcd/chaincfg"
	"github.com/ltcmweb/ltcd/chaincfg/chainhash"
	"github.com/ltcmweb/ltcd/ltcutil"
	"github.com/ltcmweb/ltcd/ltcutil/mweb"
	"github.com/ltcmweb/ltcd/ltcutil/mweb/mw"
	"github.com/ltcmweb/ltcd/wire"
)

// Fees used in TestMWOnionMath; must satisfy TestHopFeeCoversBackwardMinimum.
const testHopFeeSat = uint64(10_000)

func TestMWOnionMath(t *testing.T) {
	t.Parallel()

	privs := make([]*ecdh.PrivateKey, mlnroute.ExpectedHops)
	pubs := make([]*ecdh.PublicKey, mlnroute.ExpectedHops)
	rawHex := make([]string, mlnroute.ExpectedHops)
	for i := range privs {
		var err error
		privs[i], err = ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pubs[i] = privs[i].PublicKey()
		rawHex[i] = hex.EncodeToString(pubs[i].Bytes())
	}

	var scanKey, spendKey mw.SecretKey
	for {
		if _, err := rand.Read(scanKey[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := rand.Read(spendKey[:]); err != nil {
			t.Fatal(err)
		}
		if secretScalarOK(&scanKey) && secretScalarOK(&spendKey) {
			break
		}
	}
	kc := &mweb.Keychain{Scan: &scanKey, Spend: &spendKey}
	dest := ltcutil.NewAddressMweb(kc.Address(0), &chaincfg.MainNetParams)

	amount := uint64(100_000_000)
	req := mlnroute.Request{
		Destination: dest.String(),
		Amount:      amount,
		Route: []mlnroute.Hop{
			{Tor: "http://a.onion", FeeMinSat: testHopFeeSat, SwapX25519PubHex: rawHex[0]},
			{Tor: "http://b.onion", FeeMinSat: testHopFeeSat, SwapX25519PubHex: rawHex[1]},
			{Tor: "http://c.onion", FeeMinSat: testHopFeeSat, SwapX25519PubHex: rawHex[2]},
		},
	}
	if err := mlnroute.Validate(&req); err != nil {
		t.Fatal(err)
	}

	var blind mw.BlindingFactor
	for {
		if _, err := rand.Read(blind[:]); err != nil {
			t.Fatal(err)
		}
		if kernelScalarOK(&blind) {
			break
		}
	}
	var oid chainhash.Hash
	if _, err := rand.Read(oid[:]); err != nil {
		t.Fatal(err)
	}
	coin := &mweb.Coin{
		SpendKey: kc.SpendKey(0),
		Blind:    &blind,
		Value:    amount,
		OutputId: &oid,
	}

	o, err := buildOnionFromMLNRoute(&req, pubs, coin)
	if err != nil {
		t.Fatal(err)
	}
	if !o.VerifySig() {
		t.Fatal("VerifySig")
	}

	wi := wireFromOnionInput(o)
	commit := wi.Commitment
	stealthSum := inputStealthBase(&wi)

	cur := o
	for i := 0; i < mlnroute.ExpectedHops; i++ {
		hop, next, err := cur.Peel(privs[i])
		if err != nil {
			t.Fatalf("peel %d: %v", i, err)
		}
		commit = *peelCommitStep(&commit, hop)
		stealthSum = peelStealthStep(stealthSum, hop)
		cur = next
		if i < mlnroute.ExpectedHops-1 && hop.Output != nil {
			t.Fatalf("hop %d: unexpected output payload", i)
		}
		if i == mlnroute.ExpectedHops-1 {
			if hop.Output == nil {
				t.Fatal("last hop: missing output")
			}
			if !peelOnionsLastHopChecks(&commit, stealthSum, hop.Output) {
				t.Fatal("last hop: peelOnions-equivalent MW checks failed")
			}
		}
	}
}

func wireFromOnionInput(o *onion.Onion) wire.MwebInput {
	var wid chainhash.Hash
	copy(wid[:], o.Input.OutputId)
	var commit mw.Commitment
	copy(commit[:], o.Input.Commitment)
	var inPK, outPK mw.PublicKey
	copy(inPK[:], o.Input.InputPubKey)
	copy(outPK[:], o.Input.OutputPubKey)
	var sig mw.Signature
	copy(sig[:], o.Input.Signature)
	return wire.MwebInput{
		Features:     wire.MwebInputStealthKeyFeatureBit,
		OutputId:     wid,
		Commitment:   commit,
		InputPubKey:  &inPK,
		OutputPubKey: outPK,
		Signature:    sig,
	}
}

func peelOnionsLastHopChecks(commit2 *mw.Commitment, stealthSum *mw.PublicKey, out *wire.MwebOutput) bool {
	var msg bytes.Buffer
	out.Message.Serialize(&msg)
	if *commit2 != out.Commitment || *stealthSum != out.SenderPubKey ||
		out.RangeProof == nil ||
		!out.RangeProof.Verify(*commit2, msg.Bytes()) ||
		!out.VerifySig() {
		return false
	}
	return true
}

// TestHopFeeCoversBackwardMinimum encodes swap.go backward() fee lower bound for nOutputs=4, nNodes=3.
func TestHopFeeCoversBackwardMinimum(t *testing.T) {
	t.Parallel()
	nOutputs := uint64(4)
	nNodes := uint64(3)
	share := nOutputs * mweb.StandardOutputWeight * mweb.BaseMwebFee
	share = (share + nNodes - 1) / nNodes
	minPerNode := share + mweb.KernelWithStealthWeight*mweb.BaseMwebFee
	testSum := testHopFeeSat * mlnroute.ExpectedHops
	if testSum < minPerNode {
		t.Fatalf("test hop fee sum %d < backward minimum %d (adjust testHopFeeSat)", testSum, minPerNode)
	}
}
