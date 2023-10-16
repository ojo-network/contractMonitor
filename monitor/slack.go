package monitor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/slack-go/slack"
)

func (c *cosmwasmChecker) createLowBalanceAttachment(amount int64, balance string) slack.Attachment {
	warning := (amount <= c.warning) && (amount > c.threshold)
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", c.network, Relayer),
		Title:   fmt.Sprintf(":exclamation: %s", LowBalance),
		Color:   "danger",
		Fields: []slack.AttachmentField{
			{
				Title: "Relayer Address",
				Value: fmt.Sprintf("```%s```", c.relayerAddress),
				Short: false,
			},
			{
				Title: "Current balance",
				Value: fmt.Sprintf("```%s%s```", balance, c.denom),
				Short: true,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", c.network),
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

func (c *cosmwasmChecker) createStaleRequestIDAttachment() *slack.Attachment {
	if !(c.rateError || c.medianError || c.deviationError) {
		// no errors
		return nil
	}

	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", c.network, Relayer),
		Color:   "danger",
		Fields: []slack.AttachmentField{
			{
				Title: "Contract Address",
				Value: fmt.Sprintf("```%s```", c.contractAddress),
				Short: false,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", c.network),
				Short: false,
			},
		},
		Footer: "Monitor Bot",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	title := ":exclamation: Stale"
	if c.rateError {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Stale Request ID",
			Value: fmt.Sprintf("```%d```", c.requestID),
			Short: true,
		})

		title = fmt.Sprintf("%s %s", title, "Request")
		c.rateError = false
	}

	if c.medianError {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Stale Median ID",
			Value: fmt.Sprintf("```%d```", c.medianID),
			Short: true,
		})

		title = fmt.Sprintf("%s %s", title, "Median")

		c.medianError = false
	}

	if c.deviationError {
		attachment.Fields = append(attachment.Fields, slack.AttachmentField{
			Title: "Stale Deviation ID",
			Value: fmt.Sprintf("```%d```", c.deviationID),
			Short: true,
		})

		title = fmt.Sprintf("%s %s", title, "Deviation")

		c.deviationError = false
	}
	attachment.Title = fmt.Sprintf("%s %s", title, "ID")

	return &attachment
}

func (c *cosmwasmChecker) currentBalanceAttachment() *slack.Attachment {
	c.mut.Lock()
	defer c.mut.Unlock()
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", c.network, Relayer),
		Title:   fmt.Sprint(CurrentBalance),
		Color:   "good",
		Fields: []slack.AttachmentField{
			{
				Title: "Relayer Address",
				Value: fmt.Sprintf("```%s```", c.relayerAddress),
				Short: false,
			},
			{
				Title: "Current balance",
				Value: fmt.Sprintf("```%s%s```", c.balance, c.denom),
				Short: true,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", c.network),
				Short: true,
			},
		},
		Footer: "Monitor Bot",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	return &attachment
}

func (c *cosmwasmChecker) currentRequestIDAttachment() *slack.Attachment {
	c.mut.Lock()
	defer c.mut.Unlock()
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", c.network, Relayer),
		Title:   fmt.Sprint(RequestIDS),
		Color:   "good",
		Fields: []slack.AttachmentField{
			{
				Title: "Contract Address",
				Value: fmt.Sprintf("```%s```", c.contractAddress),
				Short: false,
			},
			{
				Title: "Current Request ID",
				Value: fmt.Sprintf("```%d```", c.requestID),
				Short: true,
			},
			{
				Title: "Current Median ID",
				Value: fmt.Sprintf("```%d```", c.medianID),
				Short: true,
			},
			{
				Title: "Current Deviation ID",
				Value: fmt.Sprintf("```%d```", c.deviationID),
				Short: true,
			},
			{
				Title: "Network",
				Value: fmt.Sprintf("```%s```", c.network),
				Short: false,
			},
		},
		Footer: "Monitor Bot",
		Ts:     json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
	}

	return &attachment
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
