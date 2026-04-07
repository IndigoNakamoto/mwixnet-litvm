package receiptstore

import (
	"database/sql"
	"fmt"
	"math/big"
	"strings"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/pkg/litvmevidence"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
)

// ReceiptRecord is the preimage plus hop receipt fields persisted for defense or accuser filing.
type ReceiptRecord struct {
	litvmevidence.EvidencePreimage
	NextHopPubkey string
	Signature     string
	SwapID        string // optional; when set, unique for lookup via GetBySwapID
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
		swap_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_hop_receipts_swap_id
		ON hop_receipts(swap_id) WHERE swap_id IS NOT NULL AND trim(swap_id) != '';
	`

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	// Migration: older DBs without swap_id
	if _, err := db.Exec(`ALTER TABLE hop_receipts ADD COLUMN swap_id TEXT`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("migrate swap_id: %w", err)
		}
	}

	return &Store{db: db}, nil
}

// SaveReceipt computes the canonical evidenceHash and inserts idempotently.
// inserted is true when a new row was written (false on duplicate evidence_hash).
func (s *Store) SaveReceipt(r ReceiptRecord) (inserted bool, err error) {
	if r.EpochID == nil {
		return false, fmt.Errorf("SaveReceipt: EpochID is required")
	}
	evidenceHash := litvmevidence.ComputeEvidenceHash(r.EvidencePreimage)
	swapID := strings.TrimSpace(r.SwapID)

	query := `
		INSERT INTO hop_receipts (
			evidence_hash, epoch_id, accuser, accused, hop_index,
			peeled_commitment, forward_hash, next_hop_pubkey, signature, swap_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))
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
		swapID,
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

// GetByEvidenceHash implements litvmevidence.ReceiptLookup.
func (s *Store) GetByEvidenceHash(hash common.Hash) (*litvmevidence.ReceiptForDefense, error) {
	query := `
		SELECT epoch_id, accuser, accused, hop_index, peeled_commitment,
		       forward_hash, next_hop_pubkey, signature
		FROM hop_receipts
		WHERE evidence_hash = ?
	`
	row := s.db.QueryRow(query, hash.Hex())

	var (
		epochStr, accuserHex, accusedHex string
		hopIndex                         int64
		peeledHex, forwardHex            string
		nextHop, sig                     string
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

	return &litvmevidence.ReceiptForDefense{
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

// GetBySwapID returns a receipt row looked up by swap_id (trimmed).
func (s *Store) GetBySwapID(swapID string) (*litvmevidence.ReceiptForDefense, error) {
	id := strings.TrimSpace(swapID)
	if id == "" {
		return nil, fmt.Errorf("swap_id empty")
	}
	query := `
		SELECT epoch_id, accuser, accused, hop_index, peeled_commitment,
		       forward_hash, next_hop_pubkey, signature
		FROM hop_receipts
		WHERE swap_id = ?
	`
	row := s.db.QueryRow(query, id)
	var (
		epochStr, accuserHex, accusedHex string
		hopIndex                         int64
		peeledHex, forwardHex            string
		nextHop, sig                     string
	)
	err := row.Scan(&epochStr, &accuserHex, &accusedHex, &hopIndex, &peeledHex, &forwardHex, &nextHop, &sig)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no receipt found for swap_id %q", id)
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
	return &litvmevidence.ReceiptForDefense{
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
