package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"
)

var (
	client    *ethclient.Client
	rpcClient *rpc.Client

	bought = false
)

var (
	wg    sync.WaitGroup
	start time.Time
)

// txData
var (
	data []byte
)

// admin variables
var (
	gasPrice             *big.Int
	maxFeePerGas         *big.Int
	maxPriorityFeePerGas *big.Int

	adminTx *types.Transaction
)

func init() {
	var err error
	client, err = ethclient.Dial(NODE_ADDRESS)
	if err != nil {
		log.Fatal(err)
	}
	
	a, err := abi.JSON(strings.NewReader(def))
	if err != nil {
		log.Fatal(err)
	}
	
	data, err = a.Pack("publicMint", big.NewInt(3))
	fmt.Println(data)
	if err != nil {
		log.Fatal(err)
	}

}

func main() {
	type2 := make(chan struct{})
	type0 := make(chan struct{})

	block2 := make(chan struct{})
	block0 := make(chan struct{})

	updateWallets()
	createTransactions()

	go listenMempool(type2, type0)
	go func() {
		for { // прослушиваем мемпул в ожидании нужной транзакции
			select {
			case <-type2:
				buy(2, false)
				if !bought {
					go monitoringBlocks(block2, block0, int(adminTx.Type()))
					bought = true
				}
			case <-type0:
				buy(0, false)
				if !bought {
					go monitoringBlocks(block2, block0, int(adminTx.Type()))
					bought = true
				}
			default:
			}
		}
	}()

loop2:
	for { // мониторим блоки на админскую тразу (интерфейс оставил такой же)
		select {
		case <-block2:
			buy(2, true)
			break loop2
		case <-block0:
			buy(0, true)
			break loop2
		}
	}
}
