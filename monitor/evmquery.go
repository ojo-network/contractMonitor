package monitor

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed evmabi/PriceFeed.json
var oracleABI string

func checkBalanceEvm(ctx context.Context, threshold, warning *big.Int, network, rpc, accountAddress string) error {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	address := common.HexToAddress(accountAddress)
	balance, err := client.BalanceAt(ctx, address, nil)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	if balance.Cmp(warning) <= 0 {
		slackchan <- createLowBalanceAttachment(balance.Cmp(threshold) > 0, balance.String(), "ETH", accountAddress, network)
	}

	return nil
}

func checkQueryEVM(network, rpc, contractAddress, _assetName string) error {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	address := common.HexToAddress(contractAddress)

	// Ensure to replace YourContractABI with your actual contract's ABI
	parsedABI, err := abi.JSON(strings.NewReader(oracleABI))
	if err != nil {
		return fmt.Errorf("Failed to parse contract ABI: %v", err)
	}

	bytesAssetName := []byte(_assetName)
	paddedAssetName := common.RightPadBytes(bytesAssetName, 32) // ETH uses 32 bytes

	result, err := parsedABI.Pack("getMedianData", paddedAssetName)
	if err != nil {
		return fmt.Errorf("Failed to pack data for function call: %v", err)
	}

	msg := ethereum.CallMsg{To: &address, Data: result}
	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("Failed to call contract: %v", err)
	}

	var medianData struct {
		AssetName   [32]byte
		ResolveTime *big.Int
		ID          *big.Int
		Values      []*big.Int
	}

	err = parsedABI.UnpackIntoInterface(&medianData, "getMedianData", output)
	if err != nil {
		return fmt.Errorf("Failed to unpack data from function call: %v", err)
	}

	check.Lock()
	defer check.Unlock()

	id := medianData.ID.Int64()
	if id <= globallist[contractAddress] {
		slackchan <- createStaleRequestIDAttachment(globallist[contractAddress], id, contractAddress, network)
		return nil
	}

	// update to latest id
	globallist[contractAddress] = id

	return nil
}
