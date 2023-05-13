package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/exec"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func readConfig() Config {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	if err = json.Unmarshal(bytes, &config); err != nil {
		log.Fatal(err)
	}

	return config
}

func writeInfoFile(nodes []string, keys []*ecdsa.PrivateKey) {
	file, err := os.Create("info.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for i, _ := range nodes {
		privateKeyBytes := crypto.FromECDSA(keys[i])
		privateKey := hex.EncodeToString(privateKeyBytes)
		address := crypto.PubkeyToAddress(keys[i].PublicKey)

		_, err := file.WriteString(fmt.Sprintf("address: %s\nprivatekey: %s\npassword: asdasdasd\n\n", address.Hex(), privateKey))
		if err != nil {
			log.Fatal(err)
		}
	}
}

type Config struct {
	Period  uint64 `json:"period"`
	ChainID int64  `json:"chainId"`
}

func main() {

	//masternode oluşturmak için eklememiz gerekmekte
	cmd := exec.Command("bootnode", "--genkey", "bootnode.key")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	config := Config{}
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	nodes := []string{"node1", "node2", "node3"}
	var keys []*ecdsa.PrivateKey
	var addresses []common.Address

	for _, node := range nodes {
		os.MkdirAll(node, os.ModePerm)
		passwordFile, err := os.Create(fmt.Sprintf("%s/password.txt", node))
		if err != nil {
			log.Fatal(err)
		}
		passwordFile.WriteString("asdasdasd")
		passwordFile.Close()

		key, err := crypto.GenerateKey()
		if err != nil {
			log.Fatal(err)
		}
		keys = append(keys, key)

		ks := keystore.NewKeyStore(fmt.Sprintf("%s/keystore", node), keystore.StandardScryptN, keystore.StandardScryptP)
		account, err := ks.ImportECDSA(key, "asdasdasd")
		if err != nil {
			log.Fatal(err)
		}

		addresses = append(addresses, account.Address)

		// Save the private key to a file
		privateKeyBytes := crypto.FromECDSA(key)
		err = ioutil.WriteFile(fmt.Sprintf("%s/keystore/privatekey.txt", node), privateKeyBytes, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	extraVanity := 32 // ExtraData field's vanity bytes size
	extraSeal := 65   // ExtraData field's seal bytes size

	// Prepare extraData field content
	extraData := make([]byte, extraVanity+len(addresses)*common.AddressLength+extraSeal)
	for i, validator := range addresses {
		copy(extraData[extraVanity+i*common.AddressLength:], validator[:])
	}

	genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             big.NewInt(config.ChainID),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			Clique: &params.CliqueConfig{
				Period: config.Period,
				Epoch:  30000,
			},
		},
		Nonce:      0x0,
		Timestamp:  0x0,
		ExtraData:  extraData,
		GasLimit:   0x1000000,
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]core.GenesisAccount{
			addresses[0]: {Balance: big.NewInt(0).Mul(big.NewInt(1e9), big.NewInt(1e18))}, // 1B ETH
			addresses[1]: {Balance: big.NewInt(0).Mul(big.NewInt(1e9), big.NewInt(1e18))}, // 1B ETH
			addresses[2]: {Balance: big.NewInt(0).Mul(big.NewInt(1e9), big.NewInt(1e18))}, // 1B ETH
		},
	}

	// Create a file to output node info
	infoFile, err := os.Create("info.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer infoFile.Close()

	startAuthRPCPort := 8090
	startHTTPPort := 8546
	startUDPPort := 30305

	for i, node := range nodes {
		// Marshal the genesis to JSON format
		genesisJSON, err := genesis.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}

		// Write the genesis to a JSON file
		err = ioutil.WriteFile(fmt.Sprintf("%s/genesis.json", node), genesisJSON, 0644)
		if err != nil {
			log.Fatal(err)
		}

		// Initialize the genesis block
		cmd := exec.Command("geth", "--datadir", fmt.Sprintf("./%s", node), "init", fmt.Sprintf("./%s/genesis.json", node))
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Log the output of the command, this will help us understand the issue
			log.Printf("geth init output: %s", output)
			log.Fatal(err)
		}

		// Write node info to file
		infoFile.WriteString(fmt.Sprintf("Node: %s\n", node))
		infoFile.WriteString(fmt.Sprintf("Address: %s\n", addresses[i].Hex()))
		infoFile.WriteString(fmt.Sprintf("PrivateKey: %x\n", crypto.FromECDSA(keys[i])))
		infoFile.WriteString("Password: asdasdasd\n\n")

		// Create start script for each node
		startScript, err := os.Create(fmt.Sprintf("%s.sh", node))
		if err != nil {
			log.Fatal(err)
		}
		defer startScript.Close()
		startScript.WriteString(fmt.Sprintf("geth --datadir ./%s --syncmode 'full' --http --http.addr '127.0.0.1' --http.port %d --http.api 'personal,eth,net,web3,txpool,miner' --http.corsdomain \"*\" --networkid %d  -nodiscover --rpc.allow-unprotected-txs --allow-insecure-unlock --miner.etherbase %s --unlock %s --password %s/password.txt --port %d --authrpc.port %d --mine", node, startHTTPPort, config.ChainID, addresses[i].Hex(), addresses[i].Hex(), node, startUDPPort, startAuthRPCPort))
		// Increment the port numbers for the next node
		startAuthRPCPort++
		startHTTPPort++
		startUDPPort++

		//masternode sh oluşturuyoruz
		bootnodeScript, err := os.Create("startBootnode.sh")
		if err != nil {
			log.Fatal(err)
		}
		defer bootnodeScript.Close()

		bootnodeScript.WriteString("#!/bin/sh\n")
		bootnodeScript.WriteString("bootnode --nodekey=bootnode.key")

		err = bootnodeScript.Chmod(0755)
		if err != nil {
			log.Fatal(err)
		}

	}
}
