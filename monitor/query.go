package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/slack-go/slack"
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

func checkBalance(threshold int64, network, denom, rpc, relayerAddress string) error {
	bal := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", rpc, relayerAddress)
	balResp, err := http.Get(bal)
	if err != nil {
		return err
	}
	defer balResp.Body.Close()

	balBody, err := io.ReadAll(balResp.Body)
	if err != nil {
		return err
	}

	var balResponse BalResponse
	if err := json.Unmarshal(balBody, &balResponse); err != nil {
		return err
	}
	for _, balance := range balResponse.Balances {
		if balance.Denom != denom {
			continue
		}

		amount, err := strconv.ParseInt(balance.Amount, 10, 64)
		if err != nil {
			return err
		}

		if amount <= threshold {
			slackchan <- createLowBalanceAttachment(balance.Amount, denom, relayerAddress, network)
		}
	}

	return nil
}

func checkQuery(network, rpc, address string) error {
	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", rpc, address, deviation)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response Response
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return err
	}

	num, err := strconv.ParseInt(response.Data.RequestID, 10, 64)
	if err != nil {
		return err
	}

	check.Lock()
	defer check.Unlock()

	requestID := globallist[address]
	globallist[address] = num
	if num <= requestID {
		slackchan <- createStaleRequestIDAttachment(num, response.Data.RequestID, address, network)
	}

	return nil
}

func createLowBalanceAttachment(balance, denom, relayerAddress, network string) slack.Attachment {
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

	return attachment
}

func createStaleRequestIDAttachment(oldRequestID int64, currentRequestID string, contractAddress, network string) slack.Attachment {
	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("*Network*: %s\n*Relayer*: %s", network, RELAYER),
		Title:   fmt.Sprintf(":exclamation: %s", StaleRequestID),
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
