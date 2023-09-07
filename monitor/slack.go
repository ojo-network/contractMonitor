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
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, Relayer),
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

func createStaleRequestIDAttachment(requestTitle, contractAddress, network string, oldRequestID, currentRequestID int64) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, Relayer),
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
				Value: fmt.Sprintf("```%d```", currentRequestID),
				Short: true,
			},
			{
				Title: "Old Request ID",
				Value: fmt.Sprintf("```%d```", oldRequestID),
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

func balanceAttachment(balance, denom, relayerAddress, network string) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, Relayer),
		Title:   fmt.Sprint(CurrentBalance),
		Color:   "good",
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

	return attachment
}

func requestIDAttachment(contractAddress, network string, currentRequestID, medianID, deviationID int64) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, Relayer),
		Title:   fmt.Sprint(RequestIDS),
		Color:   "good",
		Fields: []slack.AttachmentField{
			{
				Title: "Contract Address",
				Value: fmt.Sprintf("```%s```", contractAddress),
				Short: false,
			},
			{
				Title: "Current Request ID",
				Value: fmt.Sprintf("```%d```", currentRequestID),
				Short: true,
			},
			{
				Title: "Current Median ID",
				Value: fmt.Sprintf("```%d```", medianID),
				Short: true,
			},
			{
				Title: "Current Deviation ID",
				Value: fmt.Sprintf("```%d```", deviationID),
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

func postErr(err error) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: "An error has occurred:",
		Text:    fmt.Sprintf("event slash command error: %s", err),
		Color:   "danger",
	}

	return attachment
}

func postTimeout(network, timeout string) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: "Notification timeout",
		Text:    fmt.Sprintf("notification timeout on network %s for %s", network, timeout),
		Color:   "good",
	}

	return attachment
}
