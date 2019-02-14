package client

import (
	"github.com/bazo-blockchain/bazo-client/network"
	"github.com/bazo-blockchain/bazo-miner/protocol"
)

func getRelevantBlocks(relevantBlockHeaders []*protocol.Block) (relevantBlocks []*protocol.Block, err error) {
	for _, blockHeader := range relevantBlockHeaders {
		err := network.BlockReq(blockHeader.Hash[:])
		if err != nil {
			return nil, err
		}

		blockI, err := network.Fetch(network.BlockChan)
		if err != nil {
			return nil, err
		}

		var block *protocol.Block
		block = blockI.(*protocol.Block)
		relevantBlocks = append(relevantBlocks, block)
	}

	return relevantBlocks, nil
}

func getRelevantBlockHeaders(pubKey [64]byte) (relevantHeadersBeneficiary []*protocol.Block, relevantHeadersConfigBF []*protocol.Block) {
	for _, blockHeader := range blockHeaders {
		if blockHeader.Beneficiary == pubKey {
			relevantHeadersBeneficiary = append(relevantHeadersBeneficiary, blockHeader)
		}

		if blockHeader.NrConfigTx > 0 || (blockHeader.NrElementsBF > 0 && blockHeader.BloomFilter.Test(pubKey[:])) {
			relevantHeadersConfigBF = append(relevantHeadersConfigBF, blockHeader)
		}
	}

	return relevantHeadersBeneficiary, relevantHeadersConfigBF
}
