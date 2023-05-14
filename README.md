# Ethereum Private Network Setup ( POA )
This repository contains a Go application that sets up an Ethereum private network with three nodes and a bootnode. The application creates a new Ethereum genesis block, initializes it for each node, and creates start scripts for each node and the bootnode.

# Contact 
Contact : developer@sabriyasin.com

Website : https://satoshiturk.com 


# Prerequisites
+ Golang
+ Ethereum (geth & bootnode binaries)
+ JSON file (config.json)

# Configuration
The configuration file config.json is used to define parameters for the network. It should contain the following properties:



+ "period": The block time in seconds for the clique proof-of-stake protocol.
+ "chainId": The unique identifier for the Ethereum network.
+ "startAuthRPCPort": The starting RPC port number for the nodes.
+ "startHTTPPort": The starting HTTP port number for the nodes.
+ "startUDPPort": The starting UDP port number for the nodes.
+ "password": The password to unlock the Ethereum accounts for the nodes.
+ "bootnodeKey": The private key for the bootnode.
+ "enodePort": The TCP and UDP port for the bootnode.

 

Example
```json
{
  "period": 3,
  "chainId": 1983,
  "startAuthRPCPort": 8090,
  "startHTTPPort": 8546,
  "startUDPPort": 30303,
  "password": "password",
  "bootnodeKey": "5fa5dbb2a3e305932946666e600d1a1ac55602fcbeffbf38daa301d5345ce68f",
  "enodePort": 30301
}

```

# Running the Application
+ Install the prerequisites.
+ Clone the repository.
+ Edit the config.json file as needed.
+ Run the Go application with go run main.go.

The application will create directories for each node, generate an Ethereum account for each node, prepare and initialize a genesis block, and create start scripts.

# Output
The application will output the following:

+ A directory for each node, containing:
  + The Ethereum keystore files.
  + The password file.
  + The private key file.
  + The genesis block file (genesis.json).
  + The start script (node.sh).
+ A file info.txt containing information about each node, including the Ethereum address and private key.
+ A start script for the bootnode (startBootnode.sh).

# Starting the Network
Run the start script for each node and the bootnode in separate terminal windows. This will start each node and the bootnode, and the nodes will start mining blocks.

# Exploring the Network
Use the HTTP interface of each node to interact with the Ethereum network. You can use the Ethereum JavaScript API (web3.js) to connect to a node, unlock the Ethereum account, and send transactions or call smart contracts.