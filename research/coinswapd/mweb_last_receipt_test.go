package main

import (
	"context"
	"testing"
)

func TestMwebGetLastReceipt_snapshotsStoredJSON(t *testing.T) {
	t.Parallel()
	ss := &swapService{}
	raw := []byte(`{"epochId":"42","accuser":"0x1111111111111111111111111111111111111111","accusedMaker":"0x2222222222222222222222222222222222222222","hopIndex":1,` +
		`"peeledCommitment":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","forwardCiphertextHash":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",` +
		`"nextHopPubkey":"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f","signature":"unsigned-swap-forward-failure-v1","swapId":"sw1"}`)
	ss.receiptMu.Lock()
	ss.lastReceiptJSON = raw
	ss.lastReceiptSwapID = "sw1"
	ss.lastReceiptErrorClass = "rpc_application"
	ss.receiptMu.Unlock()

	svc := &mwebService{ss: ss}
	got, err := svc.GetLastReceipt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("nil response")
	}
	if string(got.Receipt) != string(raw) {
		t.Fatalf("receipt mismatch: %s vs %s", got.Receipt, raw)
	}
	if got.SwapID != "sw1" || got.ForwardFailureClass != "rpc_application" {
		t.Fatalf("wrapper fields: %+v", got)
	}
}
