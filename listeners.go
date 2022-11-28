package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"strings"
	"time"
)

func listenMempool(type2 chan struct{}, type0 chan struct{}) {
	var err error
	rpcClient, err = rpc.Dial(NODE_ADDRESS)
	if err != nil {
		log.Fatal(err)
	}
	defer rpcClient.Close()

	ctx := context.Background()

	chainId, err := client.ChainID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	txsHash := make(chan common.Hash)
	subscriber := gethclient.New(rpcClient)
	_, err = subscriber.SubscribePendingTransactions(context.Background(), txsHash)
	if err != nil {
		log.Fatal(err)
	}

	signer := types.LatestSignerForChainID(chainId)

	for hash := range txsHash {

		tx, _, err := client.TransactionByHash(ctx, hash)
		if err != nil {
			continue
		}
		if tx == nil || tx.To() == nil {
			continue
		}
		msg, err := tx.AsMessage(signer, nil)
		if err != nil {
			continue
		}

		equal := strings.Contains(strings.ToLower(ADMIN_DATA), common.Bytes2Hex(tx.Data()))

		if *tx.To() == common.HexToAddress(strings.ToLower(CONTRACT_ADDRESS)) && msg.From() == common.HexToAddress(strings.ToLower(ADMIN_ADDRESS)) && equal {

			fmt.Println("ADMIN TX ", tx.Hash())

			start = time.Now()
			if tx.Type() == uint8(2) {
				maxFeePerGas = tx.GasFeeCap()
				maxPriorityFeePerGas = tx.GasTipCap()
				type2 <- struct{}{}
			} else {
				gasPrice = tx.GasPrice()
				type0 <- struct{}{}
			}

			adminTx = tx
		}
	}
}

func monitoringBlocks(type2 chan struct{}, type0 chan struct{}, dst int) {
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case <-sub.Err():
			log.Fatal(err)
		case h := <-headers:
			block, err := client.BlockByHash(context.Background(), h.Hash())
			if err != nil {
				continue
			}
			for i := 0; i < len(block.Transactions()); i++ {
				if block.Transactions()[i].Hash() == adminTx.Hash() {
					switch dst {
					case 2:
						type2 <- struct{}{}
					case 0:
						type0 <- struct{}{}
					}
					return
				}
			}
		}
	}
}
