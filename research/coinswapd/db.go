package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"

	"github.com/ltcmweb/coinswapd/onion"
	"github.com/ltcsuite/ltcwallet/walletdb"
)

var coinswapOnionsBucket = []byte("coinswap-onions")
var coinswapOnionAmountsBucket = []byte("coinswap-onion-amounts")

func saveOnion(db walletdb.DB, onion *onion.Onion) error {
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket, err := tx.CreateTopLevelBucket(coinswapOnionsBucket)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(onion); err != nil {
			return err
		}
		return bucket.Put(onion.Input.Commitment, buf.Bytes())
	})
}

func saveOnionAmount(db walletdb.DB, onion *onion.Onion, amountSat uint64) error {
	if onion == nil || amountSat == 0 {
		return nil
	}
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket, err := tx.CreateTopLevelBucket(coinswapOnionAmountsBucket)
		if err != nil {
			return err
		}
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], amountSat)
		return bucket.Put(onion.Input.Commitment, buf[:])
	})
}

func loadOnionAmount(db walletdb.DB, onion *onion.Onion) (amountSat uint64, ok bool, err error) {
	if onion == nil {
		return 0, false, nil
	}
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		bucket := tx.ReadBucket(coinswapOnionAmountsBucket)
		if bucket == nil {
			return nil
		}
		v := bucket.Get(onion.Input.Commitment)
		if len(v) != 8 {
			return nil
		}
		amountSat = binary.BigEndian.Uint64(v)
		ok = true
		return nil
	})
	return amountSat, ok, err
}

func loadOnions(db walletdb.DB) (onions []*onion.Onion, err error) {
	err = walletdb.View(db, func(tx walletdb.ReadTx) error {
		bucket := tx.ReadBucket(coinswapOnionsBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var onion *onion.Onion
			dec := gob.NewDecoder(bytes.NewReader(v))
			err = dec.Decode(&onion)
			onions = append(onions, onion)
			return err
		})
	})
	return
}

func deleteOnion(db walletdb.DB, onion *onion.Onion) error {
	if onion == nil {
		return nil
	}
	if err := deleteOnionAmount(db, onion); err != nil {
		return err
	}
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket := tx.ReadWriteBucket(coinswapOnionsBucket)
		if bucket == nil {
			return nil
		}
		return bucket.Delete(onion.Input.Commitment)
	})
}

func deleteOnionAmount(db walletdb.DB, onion *onion.Onion) error {
	if onion == nil {
		return nil
	}
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		bucket := tx.ReadWriteBucket(coinswapOnionAmountsBucket)
		if bucket == nil {
			return nil
		}
		return bucket.Delete(onion.Input.Commitment)
	})
}
