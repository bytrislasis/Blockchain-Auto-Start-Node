package main

import (
	"encoding/hex"
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

func main() {
	nodes := []string{"node1", "node2", "node3"}
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
		ks := keystore.NewKeyStore(fmt.Sprintf("%s/keystore", node), keystore.StandardScryptN, keystore.StandardScryptP)
		account, err := ks.ImportECDSA(key, "asdasdasd")
		if err != nil {
			log.Fatal(err)
		}

		// Export private key
		privateKeyBytes := crypto.FromECDSA(key)
		err = ioutil.WriteFile(fmt.Sprintf("%s/keystore/privatekey.txt", node), []byte(hex.EncodeToString(privateKeyBytes)), 0644)
		if err != nil {
			log.Fatal(err)
		}

		addresses = append(addresses, account.Address)
	}

	extraData := make([]byte, 32+len(addresses)*common.AddressLength)
	for i, validator := range addresses {
		copy(extraData[32+i*common.AddressLength:], validator[:])
	}

	// Genesis dosyası oluştur ve init et
	alloc := make(map[common.Address]core.GenesisAccount)
	for _, addr := range addresses {
		alloc[addr] = core.GenesisAccount{Balance: big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)} // 10^18 is 1 ETH
	}
	genesis := &core.Genesis{
		Config: &params.ChainConfig{
			// Clique config
			Clique: &params.CliqueConfig{
				Period: 15,
				Epoch:  30000,
			},
		},
		Nonce:      0x0,
		Timestamp:  0x0,
		ExtraData:  extraData,
		GasLimit:   0x1000000,
		Difficulty: big.NewInt(1), // Difficulty should be of type *big.Int
		Alloc:      alloc,
	}

	for _, node := range nodes {
		// Genesis dosyasını her bir düğümde başlat
		genesisBytes, err := genesis.MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s/genesis.json", node), genesisBytes, 0644)
		if err != nil {
			log.Fatal(err)
		}

		cmd := exec.Command("geth", "--datadir", fmt.Sprintf("./%s", node), "init", fmt.Sprintf("./%s/genesis.json", node))
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

}
