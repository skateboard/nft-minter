package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"os"
	"strings"
)

type Config struct {
	InfuraApiUrl string `json:"infura_api_url"`
	Value int64 `json:"value"`
	WalletPrivateKey string `json:"wallet_private_key"`
	ContractAddress string `json:"contract_address"`
	GasLimit int64 `json:"gas_limit"`
	ABI string `json:"abi"`
}

func LoadConfiguration(file string) Config {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}

func GetKey(key string) (*ecdsa.PrivateKey, common.Address, error) {
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	return privateKey, fromAddress, nil
}

func checkTransactionReceipt(client *ethclient.Client, _txHash string) int {
	txHash := common.HexToHash(_txHash)
	tx, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return -1
	}

	return int(tx.Status)
}


func main()  {
	config := LoadConfiguration("config.json")
	ctx := context.Background()

	client, err := ethclient.DialContext(ctx, config.InfuraApiUrl)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(config.Value)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, fromAddress, _ := GetKey(config.WalletPrivateKey)
	fmt.Printf("Address: %s\n", fromAddress)

	toAddress := common.HexToAddress(config.ContractAddress)
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	nonce, err := client.NonceAt(ctx, fromAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	abiP, err := abi.JSON(strings.NewReader(config.ABI))
	if err != nil {
		log.Fatal(err)
	}

	data, err := abiP.Pack(
		"mint",
	)
	if err != nil {
		log.Fatal(err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    value,
		Gas:      big.NewInt(config.GasLimit).Uint64(),
		GasPrice: gasPrice,
		Data:     data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())

	for {
		transactionStatus := checkTransactionReceipt(client, signedTx.Hash().Hex())
		fmt.Printf("tx status: %d\n", transactionStatus)

		if transactionStatus == 1 {
			break
		}
	}

	fmt.Println("tx confirmed!")
}