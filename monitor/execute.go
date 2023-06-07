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
	median    = "eyJnZXRfbWVkaWFuX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="
	rate      = "eyJnZXRfcmVmIjogeyJzeW1ib2wiOiAiQVRPTSJ9fQ=="
)

type IDS struct {
	requestID   int64
	medianID    int64
	deviationID int64
}

var (
	slackchan chan slack.Attachment
	errchan   chan error
	wg        sync.WaitGroup

	LowBalance              = "Low Balance"
	StaleRateRequestID      = "No New Request id"
	StaleMedianRequestID    = "No New Median id"
	StaleDeviationRequestID = "No New Deviation id"
)

const (
	RELAYER      = "cw-relayer"
	RATE_ID      = "Rate Request id"
	MEDIAN_ID    = "Median Rate Request id"
	DEVIATION_ID = "Deviation Request id"
)

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
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cronDuration, err := time.ParseDuration(config.CronInterval)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	for network, asset := range config.AddressMap {
		rpc := config.NetworkRpc[network]
		wg.Add(1)
		newCosmwasmChecker(
			ctx,
			cronDuration,
			asset.Threshold,
			asset.WarningThreshold,
			network,
			asset.Denom,
			rpc,
			asset.ContractAddress,
			asset.RelayerAddress,
			asset.ReportMedian,
			asset.ReportDeviation,
		)
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
