package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type (
	Config struct {
		AddressMap map[string]Relayer `mapstructure:"address_map"`
		NetworkRpc map[string]string  `mapstructure:"network_rpc"`
	}

	Relayer struct {
		ContractAddress string `mapstructure:"contract_address" validate:"required"`
		RelayerAddress  string `mapstructure:"relayer_address" validate:"required"`
		Denom           string `mapstructure:"denom" validate:"required"`

		// threshold is less than warning threshold
		WarningThreshold int64 `mapstructure:"warning_threshold" validate:"required"`
		Threshold        int64 `mapstructure:"threshold" validate:"required"`

		ReportMedian    bool   `mapstructure:"report_median" validate:"required"`
		ReportDeviation bool   `mapstructure:"report_deviation" validate:"required"`
		CronInterval    string `mapstructure:"cron_interval" validate:"required"`
	}

	AccessToken struct {
		SlackToken   string
		SlackChannel string
		AppToken     string
	}
)

func ParseConfig(args []string) (*Config, *AccessToken, error) {
	godotenv.Load(".env") //nolint
	viper.SetConfigFile(args[0])
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, nil, err
	}

	token := viper.GetString("SLACK_TOKEN")
	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}

	channel := viper.GetString("SLACK_CHANNEL")
	if channel == "" {
		channel = os.Getenv("SLACK_CHANNEL")
	}

	appToken := viper.GetString("APP_TOKEN")
	if channel == "" {
		channel = os.Getenv("APP_TOKEN")
	}

	accessToken := &AccessToken{
		SlackToken:   token,
		SlackChannel: channel,
		AppToken:     appToken,
	}

	return &config, accessToken, config.validate()
}

func (c *Config) validate() error {
	// check for cron interval parse
	for _, network := range c.AddressMap {
		if _, err := time.ParseDuration(network.CronInterval); err != nil {
			return err
		}

	}
	return nil
}
