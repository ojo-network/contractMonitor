package monitor

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"strings"
	"time"
)

func NewEventService(ctx context.Context, csmService *CosmwasmService, logger zerolog.Logger, slackClient *slack.Client) error {
	client := socketmode.New(
		slackClient,
	)

	log := logger.With().Str("service", "event").Logger()
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return

			case evt := <-client.Events:
				switch evt.Type {
				case socketmode.EventTypeConnecting:
					log.Info().Msg("Connecting to Slack with Socket Mode...")
				case socketmode.EventTypeConnectionError:
					log.Err(fmt.Errorf("connection failed. Retrying")).Send()
				case socketmode.EventTypeSlashCommand:
					command, ok := evt.Data.(slack.SlashCommand)
					if !ok {
						continue
					}

					client.Ack(*evt.Request)

					err := handleSlashCommand(csmService, &command)
					if err != nil {
						slackChan <- postErr(err)
					}
				}
			}
		}
	}()

	return client.RunContext(ctx)
}

func handleSlashCommand(cms *CosmwasmService, command *slack.SlashCommand) error {
	commands := strings.Split(command.Text, " ")
	if len(commands) < 1 {
		return fmt.Errorf("no network")
	}

	network := commands[0]
	switch command.Command {
	case "/balance":
		balance, denom, relayer, err := cms.GetBalance(network)
		if err != nil {
			return err
		}

		slackChan <- balanceAttachment(balance, denom, relayer, network)

	case "/relayerstatus":
		rid, mid, did, address, err := cms.GetIDS(network)
		if err != nil {
			return err
		}

		slackChan <- requestIDAttachment(address, network, rid, mid, did)

	case "/timeout":
		if len(commands) < 2 {
			return fmt.Errorf("no timeout provided")
		}
		timeoutStr := commands[1]
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}

		return cms.SetTimeout(network, timeout)
	}
	return nil
}
