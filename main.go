package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Response struct {
	Data struct {
		Rate        string `json:"rate"`
		ResolveTime string `json:"resolve_time"`
		RequestID   string `json:"request_id"`
	} `json:"data"`
}

type Balance struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type Pagination struct {
	NextKey *string `json:"next_key"`
	Total   string  `json:"total"`
}

type BalResponse struct {
	Balances   []Balance  `json:"balances"`
	Pagination Pagination `json:"pagination"`
}

type Relayer struct {
	ContractAddress string `mapstructure:"contract_address"`
	RelayerAddress  string `mapstructure:"relayer_address"`
}

type Config struct {
	AddressMap map[string]Relayer `mapstructure:"address_map"`
	NetworkRpc map[string]string  `mapstructure:"network_rpc"`
}

const (
	deviation = "eyJnZXRfZGV2aWF0aW9uX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="
)

var errchan chan error
var check sync.Mutex
var globallist map[string]int64

func main() {

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(config)

	errchan = make(chan error, len(config.AddressMap))
	globallist = make(map[string]int64)

	for network, address := range config.AddressMap {
		globallist[address.ContractAddress] = 0
		rpc := config.NetworkRpc[network]
		go func(rpc, relayer, contractAddress string) {
			for {
				if err := checkBalance(rpc, relayer); err != nil {
					errchan <- err
				}

				err := checkQuery(rpc, contractAddress)
				if err != nil {
					errchan <- err
				}

				time.Sleep(10 * time.Second)
			}
		}(rpc, address.RelayerAddress, address.ContractAddress)
	}

	go func() {
		for err := range errchan {
			fmt.Println(err)
		}
	}()

	select {}
}

func checkBalance(rpc, relayerAddress string) error {
	bal := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", rpc, relayerAddress)
	balResp, err := http.Get(bal)
	if err != nil {
		return err
	}
	defer balResp.Body.Close()

	balBody, err := ioutil.ReadAll(balResp.Body)
	if err != nil {
		return err
	}

	var balResponse BalResponse
	if err := json.Unmarshal(balBody, &balResponse); err != nil {
		return err
	}
	for _, balance := range balResponse.Balances {
		bal, err := strconv.ParseInt(balance.Amount, 10, 64)
		if err != nil {
			return err
		}

		if bal < 10000 {
			return fmt.Errorf("bal running low")
		}
	}

	return nil
}

func checkQuery(rpc, address string) error {
	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", rpc, address, deviation)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response Response
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return err
	}

	fmt.Println(response.Data)

	num, err := strconv.ParseInt(response.Data.RequestID, 10, 64)
	if err != nil {
		return err
	}

	check.Lock()
	defer check.Unlock()

	requestID := globallist[address]
	globallist[address] = num
	if num <= requestID {
		return fmt.Errorf("request id %d did not increase", requestID)
	}

	return nil
}
