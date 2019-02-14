package cstorage

import (
	"github.com/bazo-blockchain/bazo-miner/protocol"
	"github.com/boltdb/bolt"
)

func ReadBlockHeader(hash [32]byte) (header *protocol.Block) {
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blockheaders"))
		encodedHeader := b.Get(hash[:])
		header = header.Decode(encodedHeader)

		return nil
	})

	if header == nil {
		return nil
	}

	return header
}

func ReadLastBlockHeader() (header *protocol.Block) {
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("lastblockheader"))
		cb := b.Cursor()
		_, encodedHeader := cb.First()
		header = header.Decode(encodedHeader)

		return nil
	})

	if header == nil {
		return nil
	}

	return header
}

func ReadMptProofs() (proof *protocol.MPT_Proof, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(MERKLEPATRICIAPROOF_BUCKET))
		k, _ := b.Cursor().First()
		encdoedProof := b.Get(k)
		proof = proof.Decode(encdoedProof)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return proof, nil
}
