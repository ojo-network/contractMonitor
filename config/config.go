package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type (
	Config struct {
		CronInterval string             `mapstructure:"cron_interval"`
		AddressMap   map[string]Relayer `mapstructure:"address_map"`
		NetworkRpc   map[string]string  `mapstructure:"network_rpc"`
	}

	Relayer struct {
		ContractAddress string `mapstructure:"contract_address"`
		RelayerAddress  string `mapstructure:"relayer_address"`
		Denom           string `mapstructure:"denom"`
		Threshold       int64  `mapstructure:"threshold"`
	}

	AccessToken struct {
		SlackToken   string
		SlackChannel string
	}
)

func ParseConfig(args []string) (*Config, *AccessToken, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, nil, err
	}

	viper.SetConfigFile(args[0])
	err = viper.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, nil, err
	}

	token := os.Getenv("SLACK_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL")
	accessToken := &AccessToken{
		SlackToken:   token,
		SlackChannel: channel,
	}

	return &config, accessToken, nil
}
