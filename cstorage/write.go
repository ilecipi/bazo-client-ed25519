package cstorage

import (
	"github.com/bazo-blockchain/bazo-miner/protocol"
	"github.com/boltdb/bolt"
)

func WriteBlockHeader(header *protocol.Block) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blockheaders"))
		err := b.Put(header.Hash[:], header.EncodeHeader())

		return err
	})

	return err
}

//Before saving the last block header, delete all existing entries.
func WriteLastBlockHeader(header *protocol.Block) (err error) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("lastblockheader"))
		b.ForEach(func(k, v []byte) error {
			b.Delete(k)

			return nil
		})

		return nil
	})

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("lastblockheader"))
		err := b.Put(header.Hash[:], header.EncodeHeader())

		return err
	})

	return err
}

func WriteMptProof(proof *protocol.MPT_Proof) (err error) {
	//Delete all MPT Proofs from DB, because the client only needs to send one proof
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(MERKLEPATRICIAPROOF_BUCKET))
		b.ForEach(func(k, v []byte) error {
			b.Delete(k)
			return nil
		})
		return nil
	})

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(MERKLEPATRICIAPROOF_BUCKET))
		key := proof.Hash()
		return b.Put(key[:], proof.Encode())
	})
}
