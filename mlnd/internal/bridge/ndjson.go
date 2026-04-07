package bridge

import (
	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/receiptstore"
)

// ParseReceiptLine decodes one NDJSON line into a ReceiptRecord for SaveReceipt.
func ParseReceiptLine(line []byte) (receiptstore.ReceiptRecord, error) {
	return receiptstore.ParseReceiptNDJSON(line)
}
