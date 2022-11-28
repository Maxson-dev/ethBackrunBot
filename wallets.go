package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"os"
	"strings"
)

type Wallet struct {
	privKey *ecdsa.PrivateKey
	address common.Address
	nonce   uint64

	tx  *types.DynamicFeeTx
	ltx *types.LegacyTx
}

type DepositWallet struct {
	Wallet
	raw string
}

var wallets []Wallet
var depositWallet DepositWallet

func updateWallets() {
	c, err := os.ReadFile("./privKeys.txt")
	if err != nil {
		log.Fatal(err)
	}
	keys := strings.Split(string(c), "\n")

	for _, key := range keys {
		var wallet Wallet

		priv, err := crypto.HexToECDSA(key)
		if err != nil {
			fmt.Println(err)
			continue
		}

		pub := priv.Public()
		pubECDSA := pub.(*ecdsa.PublicKey)
		from := crypto.PubkeyToAddress(*pubECDSA)

		nonce, err := client.PendingNonceAt(context.Background(), from)
		if err != nil {
			fmt.Println(err)
			continue
		}

		wallet.privKey = priv
		wallet.address = from
		wallet.nonce = nonce

		wallets = append(wallets, wallet)
	}

	priv, err := crypto.HexToECDSA(DEPOSIT_PRIVATEKEY)
	if err != nil {
		log.Fatal(err)
	}
	pub := priv.Public()
	pubECDSA := pub.(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*pubECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Fatal(err)
	}

	depositWallet.privKey = priv
	depositWallet.address = from
	depositWallet.nonce = nonce

	fmt.Println("Wallets updated")
}

func createTransactions() {

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(TX_VALUE * 1000000000000000000) // toWei
	gasLimit := uint64(GAS_LIMIT)
	contract := common.HexToAddress(CONTRACT_ADDRESS)

	for i := 0; i < len(wallets); i++ {

		wallets[i].tx = &types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     wallets[i].nonce,
			GasTipCap: nil,
			GasFeeCap: nil,
			Gas:       gasLimit,
			To:        &contract,
			Value:     value,
			Data:      data,
		}
		wallets[i].ltx = &types.LegacyTx{
			Nonce:    wallets[i].nonce,
			GasPrice: nil,
			Gas:      gasLimit,
			To:       &contract,
			Value:    value,
			Data:     data,
		}
	}

	// собираем депозит транзу
	a, err := abi.JSON(strings.NewReader(dep))
	if err != nil {
		log.Fatal(err)
	}
	depos, err := a.Pack("deposite", big.NewInt(DEPOSIT_AMOUNT))
	if err != nil {
		log.Fatal(err)
	}

	dc := common.HexToAddress(DEPOSIT_CONTRACT)
	signTx, err := types.SignNewTx(depositWallet.privKey, types.LatestSignerForChainID(chainID),
		&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     depositWallet.nonce,
			GasTipCap: big.NewInt(DEPOSIT_MIN_GWEI * 1000000000),
			GasFeeCap: big.NewInt(DEPOSIT_MAX_GWEI * 1000000000),
			Gas:       uint64(DEPOSIT_GAS_LIMIT),
			To:        &dc,
			Value:     big.NewInt(DEPOSIT_AMOUNT),
			Data:      depos,
		})
	if err != nil {
		log.Fatal(err)
	}
	b, err := signTx.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}
	depositWallet.raw = hexutil.Encode(b)

	fmt.Println(depositWallet.raw)

	fmt.Println("Transaction builded")
}
