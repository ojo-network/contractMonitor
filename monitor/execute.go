package monitor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/ojo-network/contractMonitor/config"
)

const (
	deviation = "eyJnZXRfZGV2aWF0aW9uX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="
)

var (
	slackchan  chan slack.Attachment
	errchan    chan error
	check      sync.Mutex
	globallist map[string]int64

	LowBalance     = "Low Balance"
	StaleRequestID = "No New Request id"
)

const RELAYER = "cw-relayer"

var rootCmd = &cobra.Command{
	Use:   "monitor [config-file]",
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
	config, accessToken, err := config.ParseConfig(args)
	if err != nil {
		return err
	}

	client := slack.New(accessToken.SlackToken, slack.OptionDebug(false))
	slackchan = make(chan slack.Attachment, len(config.AddressMap))

	errchan = make(chan error, len(config.AddressMap))
	globallist = make(map[string]int64)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cronDuration, err := time.ParseDuration(config.CronInterval)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	var wg sync.WaitGroup
	for network, asset := range config.AddressMap {
		globallist[asset.ContractAddress] = 0
		rpc := config.NetworkRpc[network]
		wg.Add(1)
		go func(ctx context.Context, threshold, warning int64, network, denom, rpc, relayer, contractAddress string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := checkBalance(threshold, warning, network, denom, rpc, relayer); err != nil {
						errchan <- err
					}

					err := checkQuery(network, rpc, contractAddress)
					if err != nil {
						errchan <- err
					}

					time.Sleep(cronDuration)
				}
			}
		}(ctx, asset.Threshold, asset.WarningThreshold, network, asset.Denom, rpc, asset.RelayerAddress, asset.ContractAddress)
	}

	go func() {
		for err := range errchan {
			logger.Err(err).Msg("Error in monitoring")
		}
	}()

	go func() {
		for attachment := range slackchan {
			_, timestamp, err := client.PostMessage(accessToken.SlackChannel, slack.MsgOptionAttachments(attachment))
			if err != nil {
				errchan <- err
			}

			logger.Info().Str("Posted at timestamp", timestamp).Msg("slack message posted")
		}
	}()

	trapSignal(cancel)
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("closing monitor, waiting for all routines to exit")

			wg.Wait() // waiting for all goroutines to exit
			close(errchan)
			close(slackchan)
			return nil
		}
	}
}

func trapSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT)

	go func() {
		<-sigCh
		cancel()
	}()
}
