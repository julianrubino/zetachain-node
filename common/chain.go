package common

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	SigningAlgoSecp256k1 = SigninAlgo("secp256k1")
	SigningAlgoEd25519   = SigninAlgo("ed25519")
)

// return the ChainName from a string
// if no such name exists, returns the empty chain name: ChainName_empty
func ParseChainName(chain string) ChainName {
	c := ChainName_value[chain]
	return ChainName(c)
}

type SigninAlgo string

// Chain is an alias of string , represent a block chain
//type Chain string

// Chains represent a slice of Chain
type Chains []Chain

// Equals compare two chain to see whether they represent the same chain
func (chain Chain) IsEqual(c Chain) bool {
	if chain.ChainName == c.ChainName && chain.ChainId == c.ChainId {
		return true
	}
	return false
}

func (chain Chain) IsZetaChain() bool {
	return chain.IsEqual(ZetaChain())
}
func (chain Chain) IsExternalChain() bool {
	return !chain.IsEqual(ZetaChain())
}

// bytes representations of address
// on EVM chain, it is 20Bytes
// on Bitcoin chain, it is P2WPKH address, []byte(bech32 encoded string)
func (chain Chain) EncodeAddress(b []byte) (string, error) {
	if IsEVMChain(chain.ChainId) {
		addr := ethcommon.BytesToAddress(b)
		if addr == (ethcommon.Address{}) {
			return "", fmt.Errorf("invalid EVM address")
		}
		return addr.Hex(), nil
	} else if IsBitcoinChain(chain.ChainId) {
		addrStr := string(b)
		chainParams, err := GetBTCChainParams(chain.ChainId)
		if err != nil {
			return "", err
		}
		_, err = btcutil.DecodeAddress(addrStr, chainParams)
		if err != nil {
			return "", err
		}
		return addrStr, nil
	}
	return "", fmt.Errorf("chain (%d) not supported", chain.ChainId)
}

func (chain Chain) BTCAddressFromWitnessProgram(witnessProgram []byte) (string, error) {
	chainParams, err := GetBTCChainParams(chain.ChainId)
	if err != nil {
		return "", err
	}
	address, err := btcutil.NewAddressWitnessPubKeyHash(witnessProgram, chainParams)
	if err != nil {
		return "", err
	}
	return address.EncodeAddress(), nil
}

// DecodeAddress decode the address string to bytes
func (chain Chain) DecodeAddress(addr string) ([]byte, error) {
	if IsEVMChain(chain.ChainId) {
		return ethcommon.HexToAddress(addr).Bytes(), nil
	} else if IsBitcoinChain(chain.ChainId) {
		return []byte(addr), nil
	}
	return nil, fmt.Errorf("chain (%d) not supported", chain.ChainId)
}

func IsEVMChain(chainID int64) bool {
	return chainID == 5 || // Goerli
		chainID == 80001 || // Polygon mumbai
		chainID == 97 || // BSC testnet
		chainID == 1001 || // klaytn baobab
		chainID == 1337 || // eth privnet
		chainID == 1 || // eth mainnet
		chainID == 56 || // bsc mainnet
		chainID == 137 // polygon mainnet
}

func IsEthereum(chainID int64) bool {
	return chainID == 5 || // Goerli
		chainID == 1337 || // eth privnet
		chainID == 1 // eth mainnet
}

func (chain Chain) IsKlaytnChain() bool {
	return chain.ChainId == 1001
}

func IsBitcoinChain(chainID int64) bool {
	return chainID == 18444 || // regtest
		chainID == 18332 || //testnet
		chainID == 8332 // mainnet
}

func IsEthereumChain(chainID int64) bool {
	return chainID == 1 || // eth mainnet
		chainID == 5 || // Goerli
		chainID == 1337 // eth privnet
}

// IsEmpty is to determinate whether the chain is empty
func (chain Chain) IsEmpty() bool {
	return strings.TrimSpace(chain.String()) == ""
}

// Has check whether chain c is in the list
func (chains Chains) Has(c Chain) bool {
	for _, ch := range chains {
		if ch.IsEqual(c) {
			return true
		}
	}
	return false
}

// Distinct return a distinct set of chains, no duplicates
func (chains Chains) Distinct() Chains {
	var newChains Chains
	for _, chain := range chains {
		if !newChains.Has(chain) {
			newChains = append(newChains, chain)
		}
	}
	return newChains
}

func (chains Chains) Strings() []string {
	strings := make([]string, len(chains))
	for i, c := range chains {
		strings[i] = c.String()
	}
	return strings
}

func GetChainFromChainName(chainName ChainName) *Chain {
	chains := DefaultChainsList()
	for _, chain := range chains {
		if chainName == chain.ChainName {
			return chain
		}
	}
	return nil
}

func GetChainFromChainID(chainID int64) *Chain {
	chains := DefaultChainsList()
	for _, chain := range chains {
		if chainID == chain.ChainId {
			return chain
		}
	}
	return nil
}

func GetChainNameFromChainID(chainID int64) (string, error) {
	chain := GetChainFromChainID(chainID)
	if chain == nil {
		return "", fmt.Errorf("chain %d not found", chainID)
	}
	return chain.GetChainName().String(), nil
}

func GetBTCChainParams(chainID int64) (*chaincfg.Params, error) {
	switch chainID {
	case 18444:
		return &chaincfg.RegressionNetParams, nil
	case 18332:
		return &chaincfg.TestNet3Params, nil
	case 8332:
		return &chaincfg.MainNetParams, nil
	default:
		return nil, fmt.Errorf("error chainID %d is not a Bitcoin chain", chainID)
	}
}
