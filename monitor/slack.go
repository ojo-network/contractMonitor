package monitor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/slack-go/slack"
)

func createLowBalanceAttachment(warning bool, balance, denom, relayerAddress, network string) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, RELAYER),
		Title:   fmt.Sprintf(":exclamation: %s", LowBalance),
		Color:   "danger",
		Fields: []slack.AttachmentField{
			{
				Title: "Relayer Address",
				Value: fmt.Sprintf("```%s```", relayerAddress),
				Short: false,
			},
			{
				Title: "Current balance",
				Value: fmt.Sprintf("```%s%s```", balance, denom),
				Short: true,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", network),
				Short: true,
			},
		},
		Footer: "Monitor Bot",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	if warning {
		attachment.Color = "ff9966"
	}

	return attachment
}

func createStaleRequestIDAttachment(requestTitle string, oldRequestID int64, currentRequestID int64, contractAddress, network string) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, RELAYER),
		Title:   fmt.Sprintf(":exclamation: %s", requestTitle),
		Color:   "danger",
		Fields: []slack.AttachmentField{
			{
				Title: "Contract Address",
				Value: fmt.Sprintf("```%s```", contractAddress),
				Short: false,
			},
			{
				Title: "Current Request ID",
				Value: fmt.Sprintf("```%s```", currentRequestID),
				Short: true,
			},
			{
				Title: "Old Request ID",
				Value: fmt.Sprintf("```%s```", strconv.FormatInt(oldRequestID, 10)),
				Short: true,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", network),
				Short: false,
			},
		},
		Footer: "Monitor Bot",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	return attachment
}
