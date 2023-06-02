package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
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

var rootCmd = &cobra.Command{
	Use:   "cw-relayer [config-file]",
	Args:  cobra.ExactArgs(1),
	Short: "cw-relayer monitor",
	RunE:  cwRelayerCmdHandler,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cwRelayerCmdHandler(cmd *cobra.Command, args []string) error {
	godotenv.Load(".env")
	viper.SetConfigFile(args[0])
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return err
	}

	token = os.Getenv("SLACK_TOKEN")
	channel = os.Getenv("SLACK_CHANNEL")

	client := slack.New(token, slack.OptionDebug(false))
	slackchan = make(chan slack.Attachment, len(config.AddressMap))

	errchan = make(chan error, len(config.AddressMap))
	globallist = make(map[string]int64)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cronDuration, err := time.ParseDuration(config.CronInterval)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
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
			logger.Log().Err(err).Msg("Error in monitoring")
		}
	}()

	go func() {
		for attachment := range slackchan {
			_, timestamp, err := client.PostMessage(channel, slack.MsgOptionAttachments(attachment))
			logger.Log().Str("Posted at timestamp", timestamp).Msg("slack message posted")
			if err != nil {
				errchan <- err
			}
		}
	}()

	trapSignal(cancel)
	for {
		select {
		case <-ctx.Done():
			close(errchan)
			close(slackchan)
			return nil
		}
	}
}

func trapSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		<-sigCh
		cancel()
	}()
}
