package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tyler-smith/go-bip39"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"

	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	Period           uint64 `json:"period"`
	ChainID          int64  `json:"chainId"`
	StartAuthRPCPort int    `json:"startAuthRPCPort"`
	StartHTTPPort    int    `json:"startHTTPPort"`
	StartUDPPort     int    `json:"startUDPPort"`
	Password         string `json:"password"`
	BootnodeKey      string `json:"bootnodeKey"`
	EnodePort        int    `json:"enodePort"`
}

// main function
func main() {
	// Initialize configuration
	config := initializeConfig("config.json")

	// Generate bootnode key

	fmt.Println("---------------------------- BOOTNOIDE KEY AND URL ----------------------------")
	fmt.Println(generateBootNodeKeyAndURL(config))
	fmt.Println("---------------------------- BOOTNOIDE KEY AND URL ----------------------------")

	mnemonic := "crowd buffalo odor silver close police nominee era horn steak train vibrant"

	// Generate bootnode key
	generateBootNodeKey()

	// Define nodes
	nodes := []string{"node1", "node2", "node3"}

	// Create and initialize nodes
	keys, addresses := createAndInitializeNodesWithMnemonic(nodes, config, mnemonic)

	// Prepare extra data for genesis
	extraData := prepareExtraData(addresses)

	// Prepare genesis
	genesis := prepareGenesis(config, addresses, extraData)

	// Initialize genesis for each node and create start scripts
	initializeGenesisAndCreateStartScripts(nodes, keys, addresses, genesis, config)
}

func createAndInitializeNodesWithMnemonic(nodes []string, config Config, mnemonic string) ([]*ecdsa.PrivateKey, []common.Address) {
	var keys []*ecdsa.PrivateKey
	var addresses []common.Address

	for index, node := range nodes {
		os.MkdirAll(node, os.ModePerm)
		passwordFile, err := os.Create(fmt.Sprintf("%s/password.txt", node))
		if err != nil {
			log.Fatal(err)
		}
		passwordFile.WriteString(config.Password)
		passwordFile.Close()

		key, address := generateKeyAndAddressFromMnemonic(node, config.Password, mnemonic, index)
		keys = append(keys, key)
		addresses = append(addresses, address)
	}

	return keys, addresses
}

func toECDSA(d []byte, strict bool) *ecdsa.PrivateKey {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = crypto.S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil
	}
	priv.D = new(big.Int).SetBytes(d)
	// The priv.D must < N
	if priv.D.Cmp(crypto.S256().Params().N) >= 0 {
		return nil
	}
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil
	}
	return priv
}

func generateKeyAndAddressFromMnemonic(node, password, mnemonic string, index int) (*ecdsa.PrivateKey, common.Address) {
	// Mnemonic'ten seed oluştur
	seed := bip39.NewSeed(mnemonic, "")

	// Ethereum path standardı: m/44'/60'/0'/0/index
	path := accounts.DefaultBaseDerivationPath
	path[3] = uint32(index)

	// Seed ve path kullanarak anahtar türet
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	key, _ := derivePrivateKeyForPath(masterKey, path)

	// Anahtar kullanarak keystore oluştur
	ks := keystore.NewKeyStore(fmt.Sprintf("%s/keystore", node), keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.ImportECDSA(key, password)
	if err != nil {
		log.Fatal(err)
	}

	// Save the private key to a file
	privateKeyBytes := crypto.FromECDSA(key)
	err = ioutil.WriteFile(fmt.Sprintf("%s/keystore/privatekey.txt", node), privateKeyBytes, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return key, account.Address
}

// Bu fonksiyon, belirtilen path için özel anahtar türetir.
func derivePrivateKeyForPath(master *hdkeychain.ExtendedKey, path accounts.DerivationPath) (*ecdsa.PrivateKey, error) {
	for _, n := range path {
		var err error
		master, err = master.Child(n)
		if err != nil {
			return nil, err
		}
	}
	key, err := master.ECPrivKey()
	if err != nil {
		return nil, err
	}

	privateECDSA := toECDSA(key.Serialize(), true)
	return privateECDSA, nil
}

// initializeConfig: Bu işlev, konfigürasyon dosyasını açar, içeriğini okur ve Config yapısını doldurur.
func initializeConfig(configFile string) Config {
	config := Config{}
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

// generateBootNodeKey: Bu işlev, yeni bir bootnode anahtarı oluşturur.
func generateBootNodeKey() {
	bootnodeKey := "5fa5dbb2a3e305932946666e600d1a1ac55602fcbeffbf38daa301d5345ce68f"
	err := ioutil.WriteFile("bootnode.key", []byte(bootnodeKey), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// prepareExtraData: Bu işlev, genesis bloğu için extra data'yı hazırlar.
func prepareExtraData(addresses []common.Address) []byte {
	extraVanity := 32 // ExtraData field's vanity bytes size
	extraSeal := 65   // ExtraData field's seal bytes size

	// Prepare extraData field content
	extraData := make([]byte, extraVanity+len(addresses)*common.AddressLength+extraSeal)
	for i, validator := range addresses {
		copy(extraData[extraVanity+i*common.AddressLength:], validator[:])
	}

	return extraData
}

// prepareGenesis: Bu işlev, genesis bloğunu hazırlar.
func prepareGenesis(config Config, addresses []common.Address, extraData []byte) *core.Genesis {
	return &core.Genesis{
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
}

// initializeGenesisAndCreateStartScripts: Bu işlev, her düğüm için genesis bloğunu başlatır ve başlatma scriptlerini oluşturur.
func initializeGenesisAndCreateStartScripts(nodes []string, keys []*ecdsa.PrivateKey, addresses []common.Address, genesis *core.Genesis, config Config) {
	// Create a file to output node info
	infoFile, err := os.Create("info.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer infoFile.Close()

	startAuthRPCPort := config.StartAuthRPCPort
	startHTTPPort := config.StartHTTPPort
	startUDPPort := config.StartUDPPort

	// Generate the enode URL from the bootnode key
	bootnodeKey, _ := crypto.HexToECDSA(config.BootnodeKey)
	bootnodePubKey := bootnodeKey.PublicKey
	_ = crypto.FromECDSAPub(&bootnodePubKey)
	bootnodeURL := generateBootNodeKeyAndURL(config)

	for i, node := range nodes {

		// Write genesis to a JSON file and initialize it
		writeAndInitializeGenesis(node, genesis)

		// Write node info to file
		writeNodeInfo(config, infoFile, node, keys[i], addresses[i])

		// Create start script for each node
		createStartScript(node, addresses[i], startHTTPPort, startUDPPort, startAuthRPCPort, int(config.ChainID), bootnodeURL)

		// Increment the port numbers for the next node
		startAuthRPCPort++
		startHTTPPort++
		startUDPPort++
	}

	// Create bootnode start script
	createBootNodeStartScript(bootnodeURL)
}

// writeAndInitializeGenesis: Bu işlev, genesis bloğunu bir JSON dosyasına yazar ve başlatır.
func writeAndInitializeGenesis(node string, genesis *core.Genesis) {
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
}

// writeNodeInfo: Bu işlev, düğüm bilgilerini bir dosyaya yazar.
func writeNodeInfo(config Config, infoFile *os.File, node string, key *ecdsa.PrivateKey, address common.Address) {
	infoFile.WriteString(fmt.Sprintf("Node: %s\n", node))
	infoFile.WriteString(fmt.Sprintf("Address: %s\n", address.Hex()))
	infoFile.WriteString(fmt.Sprintf("PrivateKey: %x\n", crypto.FromECDSA(key)))
	infoFile.WriteString(fmt.Sprintf("Password: %s\n", config.Password))
}

// createStartScript: Bu işlev, bir düğüm için başlatma scripti oluşturur.
func createStartScript(node string, address common.Address, startHTTPPort, startUDPPort, startAuthRPCPort, chainID int, bootnodeURL string) {
	absolutePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	startScript, err := os.Create(fmt.Sprintf("%s.sh", node))
	if err != nil {
		log.Fatal(err)
	}
	defer startScript.Close()

	ip := getLocalIP()

	startScript.WriteString(fmt.Sprintf("geth --datadir %s/%s --syncmode 'full' --http --http.addr '%s' --http.port %d --http.api 'personal,eth,net,web3,txpool,miner' --http.corsdomain \"*\" --networkid %d  --allow-insecure-unlock --miner.etherbase %s --unlock %s --password %s/%s/password.txt --port %d --authrpc.port %d --bootnodes \"%s\" --mine", absolutePath, node, ip, startHTTPPort, chainID, address.Hex(), address.Hex(), absolutePath, node, startUDPPort, startAuthRPCPort, bootnodeURL))
}

// createBootNodeStartScript: Bu işlev, bootnode için bir başlatma scripti oluşturur.
func createBootNodeStartScript(bootnodeURL string) {
	bootnodeScript, err := os.Create("startBootnode.sh")
	if err != nil {
		log.Fatal(err)
	}
	defer bootnodeScript.Close()

	bootnodeScript.WriteString("#!/bin/sh\n")
	bootnodeScript.WriteString(fmt.Sprintf("bootnode --nodekey=bootnode.key"))

	err = bootnodeScript.Chmod(0755)
	if err != nil {
		log.Fatal(err)
	}
}

// generateBootNodeKeyAndURL: Bu işlev, yeni bir bootnode anahtarı ve enode URL'si oluşturur.
func generateBootNodeKeyAndURL(config Config) string {
	bootnodeKey, err := crypto.HexToECDSA(config.BootnodeKey)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("bootnode.key", []byte(config.BootnodeKey), 0644)
	if err != nil {
		log.Fatal(err)
	}

	bootnodePubKey := bootnodeKey.PublicKey
	ip := net.ParseIP("127.0.0.1")
	bootnodeEnode := enode.NewV4(&bootnodePubKey, ip, config.EnodePort, config.EnodePort)

	return bootnodeEnode.URLv4()
}

func getLocalIP() net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}

	for _, address := range addrs {
		// IPNet tipindeki adresi alıyoruz; loopback adreslerini kontrol ediyoruz.
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
		}
	}
	return nil
}
