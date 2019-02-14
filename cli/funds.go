package cli

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/bazo-blockchain/bazo-client/network"
	"github.com/bazo-blockchain/bazo-client/util"
	"github.com/bazo-blockchain/bazo-miner/crypto"
	"github.com/bazo-blockchain/bazo-miner/p2p"
	"github.com/bazo-blockchain/bazo-miner/protocol"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli"
	"log"
)

type fundsArgs struct {
	header			int
	fromWalletFile	string
	toWalletFile	string
	toAddress		string
	amount			uint64
	fee				uint64
	txcount		    int
}

func GetFundsCommand(logger *log.Logger) cli.Command {
	return cli.Command {
		Name:	"funds",
		Usage:	"send funds from one account to another",
		Action:	func(c *cli.Context) error {
			args := &fundsArgs{
				header: 		c.Int("header"),
				fromWalletFile: c.String("from"),
				toWalletFile: 	c.String("to"),
				toAddress: 		c.String("toAddress"),
				amount: 		c.Uint64("amount"),
				fee: 			c.Uint64("fee"),
				txcount:		c.Int("txcount"),
			}

			return sendFunds(args, logger)
		},
		Flags:	[]cli.Flag {
			cli.IntFlag {
				Name: 	"header",
				Usage: 	"header flag",
				Value:	0,
			},
			cli.StringFlag {
				Name: 	"from",
				Usage: 	"load the sender's private key from `FILE`",
			},
			cli.StringFlag {
				Name: 	"to",
				Usage: 	"load the recipient's public key from `FILE`",
			},
			cli.StringFlag {
				Name: 	"toAddress",
				Usage: 	"the recipient's 128 byze public address",
			},
			cli.Uint64Flag {
				Name: 	"amount",
				Usage:	"specify the amount to send",
			},
			cli.Uint64Flag {
				Name: 	"fee",
				Usage:	"specify the fee",
				Value: 	1,
			},
			cli.IntFlag {
				Name: 	"txcount",
				Usage:	"the sender's current transaction counter",
			},
		},
	}
}

func sendFunds(args *fundsArgs, logger *log.Logger) error {
	err := args.ValidateInput()
	if err != nil {
		return err
	}

	fromPrivKey, err := crypto.ExtractECDSAKeyFromFile(args.fromWalletFile)
	if err != nil {
		return err
	}



	var toPubKey *ecdsa.PublicKey
	if len(args.toWalletFile) == 0 {
		if len(args.toAddress) == 0 {
			return errors.New(fmt.Sprintln("No recipient specified"))
		} else {
			if len(args.toAddress) != 128 {
				return errors.New(fmt.Sprintln("Invalid recipient address"))
			}

			runes := []rune(args.toAddress)
			pub1 := string(runes[:64])
			pub2 := string(runes[64:])

			toPubKey, err = crypto.GetPubKeyFromString(pub1, pub2)
			if err != nil {
				return err
			}
		}
	} else {
		toPubKey, err = crypto.ExtractECDSAPublicKeyFromFile(args.toWalletFile)
		if err != nil {
			return err
		}
	}

	fromAddress := crypto.GetAddressFromPubKey(&fromPrivKey.PublicKey)
	toAddress := crypto.GetAddressFromPubKey(toPubKey)

	////retrieve state form the network
	//currentState, err := network.StateReq(util.Config.BootstrapIpport, util.Config.ThisIpport)
	//if err != nil {
	//	return err
	//}
	//
	///*state, err := network.Fetch(network.StateChan)
	//if err != nil {
	//	return err
	//}
	//
	//var state2 map[[64]byte]*protocol.Account
	//state2 = state.(map[[64]byte]*protocol.Account)*/
	//
	////currentState := cstorage.RetrieveState()
	//
	//merklePatriciaTree, err := protocol.BuildMPT(currentState)
	//
	//if err != nil {
	//	logger.Printf("%v\n", err)
	//	return err
	//}
	//
	//proofOutOfTrie, err := protocol.CreateProver(merklePatriciaTree,fromAddress[:])
	//
	//if err != nil {
	//	logger.Printf("%v\n", err)
	//	return err
	//}
	//
	//proofAsMap, err := MemDBToMPTMap(proofOutOfTrie)
	//
	//if err != nil {
	//	logger.Printf("%v\n", err)
	//	return err
	//}
	//
	//mpt_proof := new(protocol.MPT_Proof)
	//mpt_proof.Proofs = proofAsMap

	//cstorage.WriteMptProof(mpt_proof)

	//mpt_Proof, err := cstorage.ReadMptProofs()

	tx, err := protocol.ConstrFundsTx(
		byte(args.header),
		uint64(args.amount),
		uint64(args.fee),
		uint32(args.txcount),
		fromAddress,
		toAddress,
		fromPrivKey,
		nil)

	if err != nil {
		logger.Printf("%v\n", err)
		return err
	}
	//tx.MPT_Proof = *mpt_proof

	if err := network.SendTx(util.Config.BootstrapIpport, tx, p2p.FUNDSTX_BRDCST); err != nil {
		logger.Printf("%v\n", err)
		return err
	} else {
		logger.Printf("Transaction successfully sent to network:\nTxHash: %x%v", tx.Hash(), tx)
	}

	return nil
}

func MemDBToMPTMap(proofDB *ethdb.MemDatabase)  (proof map[string][]byte, err error) {
	preliminaryMPTMap := make(map[string][]byte)

	//Iterate over db of proof and append key and values to the map in MPT_Proof
	for _, key := range proofDB.Keys(){
		retrievedValue, err := proofDB.Get(key)

		if err != nil {
			err := fmt.Sprintf("Error while retrieving value for key: %x ", string(key))
			return nil, errors.New(err)
		}

		preliminaryMPTMap[string(key)] = retrievedValue
	}

	return preliminaryMPTMap, nil
}

func (args fundsArgs) ValidateInput() error {
	if len(args.fromWalletFile) == 0 {
		return errors.New("argument missing: from")
	}

	if args.txcount < 0 {
		return errors.New("invalid argument: txcnt must be >= 0")
	}

	if len(args.toWalletFile) == 0 && len(args.toAddress) == 0 {
		return errors.New("argument missing: to or toAddess")
	}

	if len(args.toWalletFile) == 0 && len(args.toAddress) != 128 {
		return errors.New("invalid argument: toAddress")
	}

	if args.fee <= 0 {
		return errors.New("invalid argument: fee must be > 0")
	}

	if args.amount <= 0 {
		return errors.New("invalid argument: amount must be > 0")
	}

	return nil
}
