package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ltcmweb/coinswapd/mlnroute"
)

// mwebRPCStub implements the same RPC surface as mwebService for wire-name checks.
type mwebRPCStub struct{}

func (*mwebRPCStub) GetBalance(context.Context) (BalanceResult, error) {
	return BalanceResult{AvailableSat: 9, SpendableSat: 8, Detail: "ok"}, nil
}

func (*mwebRPCStub) SubmitRoute(context.Context, mlnroute.Request) (interface{}, error) {
	return map[string]bool{"accepted": true}, nil
}

func TestMWEBJSONRPCMethodNames(t *testing.T) {
	t.Parallel()
	srv := rpc.NewServer()
	if err := srv.RegisterName("mweb", &mwebRPCStub{}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(srv.ServeHTTP))
	t.Cleanup(ts.Close)

	c, err := rpc.Dial(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)

	var bal BalanceResult
	if err := c.CallContext(context.Background(), &bal, "mweb_getBalance"); err != nil {
		t.Fatalf("mweb_getBalance: %v", err)
	}
	if bal.AvailableSat != 9 || bal.SpendableSat != 8 {
		t.Fatalf("balance: %+v", bal)
	}

	req := mlnroute.Request{
		Route: []mlnroute.Hop{
			{Tor: "http://a", FeeMinSat: 1},
			{Tor: "http://b", FeeMinSat: 1},
			{Tor: "http://c", FeeMinSat: 1},
		},
		Destination: "mweb1qq",
		Amount:      10,
	}
	var submitted interface{}
	if err := c.CallContext(context.Background(), &submitted, "mweb_submitRoute", req); err != nil {
		t.Fatalf("mweb_submitRoute: %v", err)
	}
}
