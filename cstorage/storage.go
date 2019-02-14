package cstorage

import (
	"github.com/bazo-blockchain/bazo-client/util"
	"github.com/bazo-blockchain/bazo-miner/miner"
	"github.com/bazo-blockchain/bazo-miner/protocol"
	"github.com/bazo-blockchain/bazo-miner/storage"
	"github.com/boltdb/bolt"
	"log"
	"time"
)

var (
	db     *bolt.DB
	logger *log.Logger
	Buckets	[]string
)

const (
	ERROR_MSG = "Initiate storage aborted: "
	BLOCKHEADERS_BUCKET = "blockheaders"
	LASTBLOCKHEADER_BUCKET = "lastblockheader"
	MERKLEPATRICIAPROOF_BUCKET = "merklepatriciaproofs"
)

//Entry function for the storage package
func Init(dbname string) (err error) {
	logger = util.InitLogger()

	Buckets = []string {
		BLOCKHEADERS_BUCKET,
		LASTBLOCKHEADER_BUCKET,
		MERKLEPATRICIAPROOF_BUCKET,
	}

	db, err = bolt.Open(dbname, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		logger.Fatal(ERROR_MSG, err)
	}

	for _, bucket := range Buckets {
		err = storage.CreateBucket(bucket, db)
		if err != nil {
			return err
		}
	}

	/*db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucket([]byte("blockheaders"))
		if err != nil {
			return fmt.Errorf(ERROR_MSG + "Create bucket: %s", err)
		}

		return nil
	})

	db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucket([]byte("lastblockheader"))
		if err != nil {
			return fmt.Errorf(ERROR_MSG+"Create bucket: %s", err)
		}

		return nil
	})*/
	return nil
}

func RetrieveState() (state map[[64]byte]*protocol.Account)  {
	return miner.GetState()
}

func TearDown() {
	db.Close()
}
