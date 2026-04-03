package store

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/litvm"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
)

// ReceiptRecord is the preimage plus hop receipt fields persisted for defense.
type ReceiptRecord struct {
	litvm.EvidencePreimage
	NextHopPubkey string
	Signature     string
}

// Store is a SQLite-backed evidence vault.
type Store struct {
	db *sql.DB
}

// NewStore opens dbPath, applies schema, and returns a Store.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS hop_receipts (
		evidence_hash TEXT PRIMARY KEY,
		epoch_id TEXT NOT NULL,
		accuser TEXT NOT NULL,
		accused TEXT NOT NULL,
		hop_index INTEGER NOT NULL,
		peeled_commitment TEXT NOT NULL,
		forward_hash TEXT NOT NULL,
		next_hop_pubkey TEXT NOT NULL,
		signature TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SaveReceipt computes the canonical evidenceHash and inserts idempotently.
// inserted is true when a new row was written (false on duplicate evidence_hash).
func (s *Store) SaveReceipt(r ReceiptRecord) (inserted bool, err error) {
	if r.EpochID == nil {
		return false, fmt.Errorf("SaveReceipt: EpochID is required")
	}
	evidenceHash := litvm.ComputeEvidenceHash(r.EvidencePreimage)

	query := `
		INSERT INTO hop_receipts (
			evidence_hash, epoch_id, accuser, accused, hop_index,
			peeled_commitment, forward_hash, next_hop_pubkey, signature
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(evidence_hash) DO NOTHING;
	`
	res, err := s.db.Exec(query,
		evidenceHash.Hex(),
		r.EpochID.String(),
		r.Accuser.Hex(),
		r.AccusedMaker.Hex(),
		int64(r.HopIndex),
		r.PeeledCommitment.Hex(),
		r.ForwardCiphertextHash.Hex(),
		r.NextHopPubkey,
		r.Signature,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// CountReceipts returns the number of rows in hop_receipts.
func (s *Store) CountReceipts() (int64, error) {
	var n int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM hop_receipts`).Scan(&n)
	return n, err
}

// GetByEvidenceHash implements litvm.ReceiptLookup.
func (s *Store) GetByEvidenceHash(hash common.Hash) (*litvm.ReceiptForDefense, error) {
	query := `
		SELECT epoch_id, accuser, accused, hop_index, peeled_commitment,
		       forward_hash, next_hop_pubkey, signature
		FROM hop_receipts
		WHERE evidence_hash = ?
	`
	row := s.db.QueryRow(query, hash.Hex())

	var (
		epochStr, accuserHex, accusedHex string
		hopIndex                        int64
		peeledHex, forwardHex           string
		nextHop, sig                    string
	)
	err := row.Scan(&epochStr, &accuserHex, &accusedHex, &hopIndex, &peeledHex, &forwardHex, &nextHop, &sig)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no receipt found for evidenceHash %s", hash.Hex())
		}
		return nil, err
	}

	epochID, ok := new(big.Int).SetString(epochStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid epoch_id decimal %q", epochStr)
	}
	if hopIndex < 0 || hopIndex > 255 {
		return nil, fmt.Errorf("invalid hop_index %d", hopIndex)
	}

	return &litvm.ReceiptForDefense{
		EpochID:               epochID,
		Accuser:               common.HexToAddress(accuserHex),
		AccusedMaker:          common.HexToAddress(accusedHex),
		HopIndex:              uint8(hopIndex),
		PeeledCommitment:      common.HexToHash(peeledHex),
		ForwardCiphertextHash: common.HexToHash(forwardHex),
		NextHopPubkey:         nextHop,
		Signature:             sig,
	}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
