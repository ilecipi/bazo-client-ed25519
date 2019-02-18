package client

import (
	"fmt"
	"github.com/bazo-blockchain/bazo-client/cstorage"
	"github.com/bazo-blockchain/bazo-client/network"
	"github.com/bazo-blockchain/bazo-miner/miner"
	"github.com/bazo-blockchain/bazo-miner/p2p"
	"github.com/bazo-blockchain/bazo-miner/protocol"
)

var (
	//All blockheaders of the whole chain
	blockHeaders []*protocol.Block

	activeParameters miner.Parameters

	UnsignedContractTx    = make(map[[32]byte]*protocol.ContractTx)
	UnsignedConfigTx = make(map[[32]byte]*protocol.ConfigTx)
	UnsignedFundsTx  = make(map[[32]byte]*protocol.FundsTx)
	SignedIotTx = make(map[[32]byte]*protocol.IotTx)
)

//Update allBlockHeaders to the latest header. Start listening to broadcasted headers after.
func Sync() {
	loadBlockHeaders()
	go incomingBlockHeaders()
}

func loadBlockHeaders() {
	var last *protocol.Block

	//youngest = fetchBlockHeader(nil)
	if last = cstorage.ReadLastBlockHeader(); last != nil {
		var loaded []*protocol.Block
		loaded = loadDB(last, [32]byte{}, loaded)
		blockHeaders = append(blockHeaders, loaded...)
	}

	//The client is up to date with the network and can start listening for incoming headers.
	network.Uptodate = true
}

func incomingBlockHeaders() {
	for {
		blockHeaderIn := <-network.BlockHeaderIn

		var last *protocol.Block
		var lastHash [32]byte

		//Get the last header in the blockHeaders array. Its hash is relevant for appending the incoming header or the abort condition for recursive header fetching.
		if len(blockHeaders) > 0 {
			last = blockHeaders[len(blockHeaders)-1]
			lastHash = last.Hash
		} else {
			lastHash = [32]byte{}
		}

		//The incoming block header is already the last saved in the array.
		if blockHeaderIn.Hash == lastHash {
			continue
		}

		//The client is out of sync. Header cannot be appended to the array. The client must sync first.
		if last == nil || blockHeaderIn.PrevHash != lastHash {
			//Set the uptodate flag to false in order to avoid listening to new incoming block headers.
			network.Uptodate = false

			var loaded []*protocol.Block

			if last == nil || len(blockHeaders) <= 100 {
				blockHeaders = []*protocol.Block{}
				loaded = loadNetwork(blockHeaderIn, [32]byte{}, loaded)
			} else {
				//Remove the last 100 headers. This is precaution if the array contains rolled back blocks.
				blockHeaders = blockHeaders[:len(blockHeaders)-100]
				loaded = loadNetwork(blockHeaderIn, blockHeaders[len(blockHeaders)-1].Hash, loaded)
			}

			blockHeaders = append(blockHeaders, loaded...)
			cstorage.WriteLastBlockHeader(blockHeaders[len(blockHeaders)-1])

			network.Uptodate = true
		} else if blockHeaderIn.PrevHash == lastHash {
			saveAndLogBlockHeader(blockHeaderIn)

			blockHeaders = append(blockHeaders, blockHeaderIn)
			cstorage.WriteLastBlockHeader(blockHeaderIn)
		}
	}
}

func fetchBlockHeader(blockHash []byte) (blockHeader *protocol.Block) {
	var errormsg string
	if blockHash != nil {
		errormsg = fmt.Sprintf("Loading header %x failed: ", blockHash[:8])
	}

	err := network.BlockHeaderReq(blockHash[:])
	if err != nil {
		logger.Println(errormsg + err.Error())
		return nil
	}

	blockHeaderI, err := network.Fetch(network.BlockHeaderChan)
	if err != nil {
		logger.Println(errormsg + err.Error())
		return nil
	}

	blockHeader = blockHeaderI.(*protocol.Block)

	logger.Printf("Fetch header with height %v\n", blockHeader.Height)

	return blockHeader
}

func loadDB(last *protocol.Block, abort [32]byte, loaded []*protocol.Block) []*protocol.Block {
	var ancestor *protocol.Block

	if last.PrevHash != abort {
		if ancestor = cstorage.ReadBlockHeader(last.PrevHash); ancestor == nil {
			logger.Fatal()
		}

		loaded = loadDB(ancestor, abort, loaded)
	}

	logger.Printf("Header %x with height %v loaded from DB\n",
		last.Hash[:8],
		last.Height)

	loaded = append(loaded, last)

	return loaded
}

func loadNetwork(last *protocol.Block, abort [32]byte, loaded []*protocol.Block) []*protocol.Block {
	var ancestor *protocol.Block
	if ancestor = fetchBlockHeader(last.PrevHash[:]); ancestor == nil {
		for ancestor == nil {
			logger.Printf("Try to fetch header %x with height %v again\n", last.Hash[:8], last.Height)
			ancestor = fetchBlockHeader(last.PrevHash[:])
		}
	}

	if last.PrevHash != abort {
		loaded = loadNetwork(ancestor, abort, loaded)
	}

	saveAndLogBlockHeader(last)

	loaded = append(loaded, last)

	return loaded
}

func saveAndLogBlockHeader(blockHeader *protocol.Block) {
	cstorage.WriteBlockHeader(blockHeader)
	logger.Printf("Header %x with height %v loaded from network\n",
		blockHeader.Hash[:8],
		blockHeader.Height)
}

func getState(acc *Account, lastTenTx []*FundsTxJson) (err error) {
	//Get blocks if the Acc address:
	//* sent funds
	//* received funds * is block's beneficiary
	//* nr of configTx in block is > 0 (in order to maintain params in light-client)

	relevantHeadersBeneficiary, relevantHeadersConfigBF := getRelevantBlockHeaders(acc.Address)

	acc.Balance += activeParameters.Block_reward * uint64(len(relevantHeadersBeneficiary))

	relevantBlocks, err := getRelevantBlocks(relevantHeadersConfigBF)
	for _, block := range relevantBlocks {
		if block != nil {
			//Balance funds and collect fee
			for _, txHash := range block.FundsTxData {
				err := network.TxReq(p2p.FUNDSTX_REQ, txHash)
				if err != nil {
					return err
				}

				txI, err := network.Fetch(network.FundsTxChan)
				if err != nil {
					return err
				}

				tx := txI.(protocol.Transaction)
				fundsTx := txI.(*protocol.FundsTx)

				if fundsTx.From == acc.Address || fundsTx.To == acc.Address || block.Beneficiary == acc.Address {
					//Validate tx
					if err := validateTx(block, tx, txHash); err != nil {
						return err
					}

					if fundsTx.From == acc.Address {
						//If Acc is no root, balance funds
						if !acc.IsRoot {
							acc.Balance -= fundsTx.Amount
							acc.Balance -= fundsTx.Fee
						}

						acc.TxCnt += 1
					}

					if fundsTx.To == acc.Address {
						acc.Balance += fundsTx.Amount

						put(lastTenTx, ConvertFundsTx(fundsTx, "verified"))
					}

					if block.Beneficiary == acc.Address {
						acc.Balance += fundsTx.Fee
					}
				}
			}

			//Update config parameters and collect fee
			for _, txHash := range block.ConfigTxData {
				err := network.TxReq(p2p.CONFIGTX_REQ, txHash)
				if err != nil {
					return err
				}

				txI, err := network.Fetch(network.ConfigTxChan)
				if err != nil {
					return err
				}

				tx := txI.(protocol.Transaction)
				configTx := txI.(*protocol.ConfigTx)

				configTxSlice := []*protocol.ConfigTx{configTx}

				if block.Beneficiary == acc.Address {
					//Validate tx
					if err := validateTx(block, tx, txHash); err != nil {
						return err
					}

					acc.Balance += configTx.Fee
				}

				miner.CheckAndChangeParameters(&activeParameters, &configTxSlice)
			}

			//TODO stakeTx

		}
	}

	return nil
}
