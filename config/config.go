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
	godotenv.Load(".env")
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

	accessToken := &AccessToken{
		SlackToken:   token,
		SlackChannel: channel,
	}

	return &config, accessToken, nil
}
