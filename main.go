package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

const (
	deviation = "eyJnZXRfZGV2aWF0aW9uX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="
)

var (
	slackchan  chan slack.Attachment
	errchan    chan error
	check      sync.Mutex
	globallist map[string]int64
	token      string
	channel    string

	LowBalance     = "Low Balance"
	StaleRequestID = "No New Request id"
)

const RELAYER = "cw-relayer"

func main() {
	godotenv.Load(".env")
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
		panic(err)
	}

	token = os.Getenv("SLACK_TOKEN")
	channel = os.Getenv("SLACK_CHANNEL")

	client := slack.New(token, slack.OptionDebug(false))
	slackchan = make(chan slack.Attachment, len(config.AddressMap))
	errchan = make(chan error, len(config.AddressMap))

	globallist = make(map[string]int64)
	cronDuration, err := time.ParseDuration(config.CronInterval)
	if err != nil {
		panic(err)
	}

	//ctx, cancel := context.WithCancel(context.Background())
	for network, asset := range config.AddressMap {
		globallist[asset.ContractAddress] = 0
		rpc := config.NetworkRpc[network]
		go func(threshold int64, network, denom, rpc, relayer, contractAddress string) {
			for {
				if err := checkBalance(threshold, network, denom, rpc, relayer); err != nil {
					errchan <- err
				}

				err := checkQuery(network, rpc, contractAddress)
				if err != nil {
					errchan <- err
				}

				time.Sleep(cronDuration)
			}
		}(asset.Threshold, network, asset.Denom, rpc, asset.RelayerAddress, asset.ContractAddress)
	}

	go func() {
		for err := range errchan {
			fmt.Println(err)
		}
	}()

	go func() {
		for attachment := range slackchan {
			_, _, err := client.PostMessage(channel, slack.MsgOptionAttachments(attachment))
			if err != nil {
				errchan <- err
			}
		}
	}()

	select {}
}
