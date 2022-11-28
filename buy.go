package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/metachris/flashbotsrpc"
	"log"
	"math/big"
	"time"
)

func buy(tp int, boost bool) {

	// увеличиваем газ до фиксированного значения
	if boost {
		switch tp {
		case 2:
			maxFeePerGas = big.NewInt(MAX_GWEI * 1000000000)
			maxPriorityFeePerGas = big.NewInt(MIN_GWEI * 1000000000)
		case 0:
			gasPrice = big.NewInt(MIN_GWEI * 1000000000)
		}
	}

	for i := 0; i < len(wallets); i++ {
		switch tp {
		case 2:
			wallets[i].tx.GasFeeCap = maxFeePerGas
			wallets[i].tx.GasTipCap = maxPriorityFeePerGas
		case 0:
			wallets[i].ltx.GasPrice = gasPrice
		}
		w := wallets[i]

		wg.Add(1)
		go func() {
			defer wg.Done()
			switch tp {
			case 2:
				tx := types.NewTx(w.tx)
				signTx, err := types.SignTx(tx, types.LatestSignerForChainID(tx.ChainId()), w.privKey)
				if err != nil {
					fmt.Println(err)
				}
				err = client.SendTransaction(context.Background(), signTx)
				if err != nil {
					fmt.Println(err)
				}
			case 0:
				tx := types.NewTx(w.ltx)
				signTx, err := types.SignTx(tx, types.LatestSignerForChainID(tx.ChainId()), w.privKey)
				if err != nil {
					fmt.Println(err)
				}
				err = client.SendTransaction(context.Background(), signTx)
				if err != nil {
					fmt.Println(err)
				}
			}
		}()
	}
	wg.Wait()
	sendFlashBotBundle(tp)
	fmt.Println(time.Since(start))
}

func sendFlashBotBundle(tp int) {
	var bundle []string
	for i := 0; i < len(wallets); i++ {
		switch tp {
		case 2:
			wallets[i].tx.GasFeeCap = big.NewInt(maxFeePerGas.Int64() + maxFeePerGas.Int64()/100*10)
			wallets[i].tx.GasTipCap = big.NewInt(maxPriorityFeePerGas.Int64() + maxPriorityFeePerGas.Int64()/100*10)
			tx := types.NewTx(wallets[i].tx)
			signTx, err := types.SignTx(tx, types.LatestSignerForChainID(tx.ChainId()), wallets[i].privKey)
			if err != nil {
				fmt.Println(err)
				continue
			}
			b, err := signTx.MarshalBinary()
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(hexutil.Encode(b))
			bundle = append(bundle, hexutil.Encode(b))
		case 0:
			wallets[i].ltx.GasPrice = big.NewInt(gasPrice.Int64() + gasPrice.Int64()/100*10)
			tx := types.NewTx(wallets[i].ltx)
			signTx, err := types.SignTx(tx, types.LatestSignerForChainID(tx.ChainId()), wallets[i].privKey)
			if err != nil {
				fmt.Println(err)
				continue
			}
			b, err := signTx.MarshalBinary()
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(hexutil.Encode(b))
			bundle = append(bundle, hexutil.Encode(b))
		}
	}
	b, err := adminTx.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(hexutil.Encode(b))
	bundle = append(bundle, hexutil.Encode(b))
	bundle = append(bundle, depositWallet.raw)

	h, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	rpc := flashbotsrpc.New("https://rpc.flashbots.net")
	for i := 0; i < 1; i++ {
		n := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			stat, _ := rpc.FlashbotsGetUserStats(depositWallet.privKey, uint64(h.Number.Int64()))
			fmt.Printf(`IS HIGH PRIORITY: %t
AllTimeMinerPayments: %s
AllTimeGasSimulated: %s`, stat.IsHighPriority, stat.AllTimeMinerPayments, stat.AllTimeGasSimulated)

			callBundleParams := flashbotsrpc.FlashbotsCallBundleParam{
				Txs:              bundle,
				BlockNumber:      hexutil.EncodeUint64(uint64(h.Number.Int64() + 1 + int64(n))),
				StateBlockNumber: "latest",
			}

			res, err := rpc.FlashbotsCallBundle(depositWallet.privKey, callBundleParams)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Simulation ", res.BundleHash)

			bundleArgs := flashbotsrpc.FlashbotsSendBundleRequest{
				Txs:         bundle,
				BlockNumber: hexutil.EncodeUint64(uint64(h.Number.Int64() + 1 + int64(n))),
			}
			fmt.Println("Bundle send to ", bundleArgs.BlockNumber)
			_, err = rpc.FlashbotsSendBundle(depositWallet.privKey, bundleArgs)
			if err != nil {
				fmt.Println(err)
				return
			}
		}()
	}
	wg.Wait()
}
