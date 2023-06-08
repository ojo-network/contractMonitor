package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type (
	Response struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}

	Balance struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}

	Pagination struct {
		NextKey *string `json:"next_key"`
		Total   string  `json:"total"`
	}

	BalResponse struct {
		Balances   []Balance  `json:"balances"`
		Pagination Pagination `json:"pagination"`
	}
)

type cosmwasmChecker struct {
	threshold       int64
	warning         int64
	network         string
	denom           string
	rpc             string
	contractAddress string
	relayerAddress  string
	reportMedian    bool
	reportDeviation bool
	requestID       int64
	deviationID     int64
	medianID        int64
	logger          zerolog.Logger
}

func newCosmwasmChecker(
	ctx context.Context,
	duration time.Duration,
	threshold int64,
	warning int64,
	network string,
	denom string,
	rpc string,
	contractAddress string,
	relayerAddress string,
	reportMedian bool,
	reportDeviation bool,
	logger zerolog.Logger,
) {
	checker := &cosmwasmChecker{
		threshold:       threshold,
		warning:         warning,
		network:         network,
		denom:           denom,
		rpc:             rpc,
		contractAddress: contractAddress,
		relayerAddress:  relayerAddress,
		reportMedian:    reportMedian,
		reportDeviation: reportDeviation,
		logger:          logger,
	}

	go checker.startCron(ctx, duration)
}

func (c *cosmwasmChecker) startCron(ctx context.Context, duration time.Duration) {
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return

		default:
			err := c.checkBalance()
			if err != nil {
				c.logger.Err(err).
					Str("relayer", c.relayerAddress).
					Str("contract", c.contractAddress).
					Str("network", c.network).
					Msg("Error in querying balance")
			}

			err = c.checkQuery(ctx)
			if err != nil {
				c.logger.Err(err).
					Str("relayer", c.relayerAddress).
					Str("contract", c.contractAddress).
					Str("network", c.network).
					Msg("Error in querying request ids")
			}

			time.Sleep(duration)
		}
	}
}

func (c *cosmwasmChecker) checkBalance() error {
	bal := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", c.rpc, c.relayerAddress)
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
		if balance.Denom != c.denom {
			continue
		}

		amount, err := strconv.ParseInt(balance.Amount, 10, 64)
		if err != nil {
			return err
		}

		if amount <= c.warning {
			slackchan <- createLowBalanceAttachment(
				(amount <= c.warning) && (amount > c.threshold),
				balance.Amount,
				c.denom,
				c.relayerAddress,
				c.network,
			)
		}
	}

	return nil
}

func (c *cosmwasmChecker) checkQuery(ctx context.Context) error {
	g, _ := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			num, err := c.returnLatestID(rate)
			if err != nil {
				return err
			}

			if num <= c.requestID {
				slackchan <- createStaleRequestIDAttachment(
					StaleRateRequestID,
					c.requestID,
					num,
					c.contractAddress,
					c.network,
				)
				return nil
			}

			c.requestID = num
			return nil
		},
	)

	if c.reportDeviation {
		g.Go(
			func() error {
				num, err := c.returnLatestID(deviation)
				if err != nil {
					return err
				}

				if num <= c.deviationID {
					slackchan <- createStaleRequestIDAttachment(
						StaleDeviationRequestID,
						c.deviationID,
						num,
						c.contractAddress,
						c.network,
					)
					return nil
				}

				// update to latest id
				c.deviationID = num
				return nil
			},
		)
	}

	if c.reportMedian {
		g.Go(
			func() error {
				num, err := c.returnLatestID(median)
				if err != nil {
					return err
				}

				if num <= c.medianID {
					slackchan <- createStaleRequestIDAttachment(
						StaleMedianRequestID,
						c.deviationID,
						num,
						c.contractAddress,
						c.network,
					)
					return nil
				}

				// update to latest id
				c.medianID = num
				return nil
			},
		)
	}

	return g.Wait()
}

func (c *cosmwasmChecker) returnLatestID(request string) (int64, error) {
	url := fmt.Sprintf("%s/cosmwasm/wasm/v1/contract/%s/smart/%s", c.rpc, c.contractAddress, request)
	resp, err := http.Get(url)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	var response Response
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return -1, err
	}

	return strconv.ParseInt(response.Data.RequestID, 10, 64)
}
