package main

import (
	"encoding/json"
	"testing"
)

func TestGoldenReceiptAccusedMaker(t *testing.T) {
	t.Parallel()
	addr := "0x3c44cdddb6a900fa2b585dd299e03d12fa4293bc"

	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "empty route falls back",
			json: `{"route":[],"destination":"x","amount":1}`,
			want: goldenReceiptAccusedFallback,
		},
		{
			name: "no operator falls back",
			json: `{"route":[{"tor":"http://a","feeMinSat":1}],"destination":"x","amount":1}`,
			want: goldenReceiptAccusedFallback,
		},
		{
			name: "valid operator lowercases",
			json: `{"route":[{"tor":"http://a","feeMinSat":1,"operator":"` + addr + `"}],"destination":"x","amount":1}`,
			want: addr,
		},
		{
			name: "operator without 0x prefix",
			json: `{"route":[{"tor":"http://a","feeMinSat":1,"operator":"3c44cdddb6a900fa2b585dd299e03d12fa4293bc"}],"destination":"x","amount":1}`,
			want: addr,
		},
		{
			name: "invalid hex falls back",
			json: `{"route":[{"tor":"http://a","feeMinSat":1,"operator":"not-an-address"}],"destination":"x","amount":1}`,
			want: goldenReceiptAccusedFallback,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var sr submitRouteBody
			if err := json.Unmarshal([]byte(tt.json), &sr); err != nil {
				t.Fatal(err)
			}
			got := goldenReceiptAccusedMaker(sr)
			if got != tt.want {
				t.Fatalf("goldenReceiptAccusedMaker() = %q, want %q", got, tt.want)
			}
		})
	}
}
