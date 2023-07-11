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

// base64 encoding for queries
const (
	// {"get_deviation_ref": {"symbol": "ATOM"}}
	deviation = "eyJnZXRfZGV2aWF0aW9uX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="

	// {"get_median_ref": {"symbol": "ATOM"}}
	median = "eyJnZXRfbWVkaWFuX3JlZiI6IHsic3ltYm9sIjogIkFUT00ifX0="

	// {"get_ref": {"symbol": "ATOM"}}
	rate = "eyJnZXRfcmVmIjogeyJzeW1ib2wiOiAiQVRPTSJ9fQ=="
)

type IDS struct {
	requestID   int64
	medianID    int64
	deviationID int64
}

var (
	slackChan chan slack.Attachment
	wg        sync.WaitGroup
)

const (
	StaleRateRequestID      = "No New Request id"
	StaleMedianRequestID    = "No New Median id"
	StaleDeviationRequestID = "No New Deviation id"
	LowBalance              = "Low Balance"
	Relayer                 = "Relayer"
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

	rootCmd.AddCommand(getVersionCmd())
}

func cwRelayerCmdHandler(cmd *cobra.Command, args []string) error {
	config, accessToken, err := config.ParseConfig(args)
	if err != nil {
		return err
	}

	client := slack.New(accessToken.SlackToken, slack.OptionDebug(false))
	slackChan = make(chan slack.Attachment, len(config.AddressMap))

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	if err != nil {
		return err
	}

	logger.Info().Msg("Contract monitor starting...")

	ctx, cancel := context.WithCancel(cmd.Context())
	for network, asset := range config.AddressMap {
		rpc := config.NetworkRpc[network]
		cronDuration, _ := time.ParseDuration(config.AddressMap[network].CronInterval)

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
			logger,
		)

		logger.Info().Str("network", network).
			Str("relayer address", asset.RelayerAddress).
			Str("contract address", asset.ContractAddress).
			Msg("monitoring")
	}

	go func() {
		for attachment := range slackChan {
			_, timestamp, err := client.PostMessage(accessToken.SlackChannel, slack.MsgOptionAttachments(attachment))
			if err != nil {
				logger.Err(err).Msg("error posting slack message")
			}

			logger.Info().Str("Posted at timestamp", timestamp).Msg("slack message posted")
		}
	}()

	trapSignal(cancel)
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("closing monitor, waiting for all monitors to exit")

			wg.Wait()
			close(slackChan)
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
