package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/ojo-network/contractMonitor/config"
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
	balance         string
	rpc             string
	contractAddress string
	relayerAddress  string
	reportMedian    bool
	reportDeviation bool
	requestID       int64
	deviationID     int64
	medianID        int64
	timeout         time.Time
	logger          zerolog.Logger
	mut             sync.Mutex
}

type CosmwasmService struct {
	services map[string]*cosmwasmChecker
}

func StartCosmwasmServices(ctx context.Context, logger zerolog.Logger, config config.Config) *CosmwasmService {
	csLogger := logger.With().Str("module", "cosmwasm-service").Logger()
	cms := CosmwasmService{
		services: make(map[string]*cosmwasmChecker),
	}
	for network, asset := range config.AddressMap {
		rpc := config.NetworkRpc[network]
		cronDuration, _ := time.ParseDuration(config.AddressMap[network].CronInterval)
		wg.Add(1)
		service := newCosmwasmChecker(
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

		cms.services[network] = service

		csLogger.Info().Str("network", network).
			Str("relayer address", asset.RelayerAddress).
			Str("contract address", asset.ContractAddress).
			Msg("monitoring")
	}

	return &cms
}

func (cws *CosmwasmService) GetBalance(network string) (balance, denom, relayerAddress string, err error) {
	service, found := cws.services[network]
	if !found {
		err = fmt.Errorf("network not found")
		return
	}

	balance, denom, relayerAddress = service.GetBalance()
	return
}

func (cws *CosmwasmService) GetIDS(network string) (rate int64, median int64, deviation int64, contractAddress string, err error) {
	service, found := cws.services[network]
	if !found {
		err = fmt.Errorf("network not found")
		return
	}

	rate, median, deviation = service.GetIds()
	contractAddress = service.contractAddress
	return
}

func (cws *CosmwasmService) SetTimeout(network string, timeout time.Duration) (err error) {
	service, found := cws.services[network]
	if !found {
		err = fmt.Errorf("network not found")
		return
	}

	service.mut.Lock()
	defer service.mut.Unlock()
	service.timeout = time.Now().Add(timeout)
	return
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
) *cosmwasmChecker {
	checker := &cosmwasmChecker{
		threshold:       threshold,
		warning:         warning,
		network:         network,
		denom:           denom,
		timeout:         time.Now(),
		rpc:             rpc,
		contractAddress: contractAddress,
		relayerAddress:  relayerAddress,
		reportMedian:    reportMedian,
		reportDeviation: reportDeviation,
		logger:          logger,
	}

	go checker.startCron(ctx, duration)

	return checker
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

func (c *cosmwasmChecker) GetBalance() (string, string, string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.balance, c.denom, c.relayerAddress
}

func (c *cosmwasmChecker) GetIds() (int64, int64, int64) {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.requestID, c.deviationID, c.medianID
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

		// update latest balance
		c.balance = balance.Amount

		if amount <= c.warning {
			if time.Now().After(c.timeout) {
				slackChan <- createLowBalanceAttachment(
					(amount <= c.warning) && (amount > c.threshold),
					balance.Amount,
					c.denom,
					c.relayerAddress,
					c.network,
				)
			}
		}
	}

	return nil
}

func (c *cosmwasmChecker) checkQuery(ctx context.Context) error {
	g, _ := errgroup.WithContext(ctx)
	post := time.Now().After(c.timeout)
	g.Go(
		func() error {
			num, err := c.returnLatestID(rate)
			if err != nil {
				return err
			}

			if num <= c.requestID {
				if post {
					slackChan <- createStaleRequestIDAttachment(
						StaleRateRequestID,
						c.contractAddress,
						c.network,
						c.requestID,
						num,
					)
					return nil
				}
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
					if post {
						slackChan <- createStaleRequestIDAttachment(
							StaleDeviationRequestID,
							c.contractAddress,
							c.network,
							c.deviationID,
							num,
						)
						return nil
					}
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
				if post {
					if num <= c.medianID {
						slackChan <- createStaleRequestIDAttachment(
							StaleMedianRequestID,
							c.contractAddress,
							c.network,
							c.medianID,
							num,
						)
						return nil
					}
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
