package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/davidseybold/unifi-technitium-syncer/syncer"
	"github.com/davidseybold/unifi-technitium-syncer/technitium"
	"github.com/davidseybold/unifi-technitium-syncer/unifi"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Config struct {
	UnifiAPIURL        string `mapstructure:"UNIFI_API_URL"`
	UnifiAPIKey        string `mapstructure:"UNIFI_API_KEY"`
	UnifiSiteID        string `mapstructure:"UNIFI_SITE_ID"`
	TechnitiumAPIURL   string `mapstructure:"TECHNITIUM_API_URL"`
	TechnitiumAPIToken string `mapstructure:"TECHNITIUM_API_TOKEN"`
	SyncZone           string `mapstructure:"SYNC_ZONE"`
}

func LoadConfig() (config Config, err error) {
	viper.AddConfigPath(".")
	viper.SetConfigName("unifi-technitium-sync")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return config, err
}

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
}

func run(ctx context.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	runId := uuid.New().String()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger = logger.With("runId", runId)

	unifiClient := unifi.NewClient(config.UnifiAPIURL, config.UnifiAPIKey, config.UnifiSiteID)
	technitiumClient := technitium.NewClient(config.TechnitiumAPIURL, config.TechnitiumAPIToken)

	s := syncer.New(unifiClient, technitiumClient, config.SyncZone, logger)

	return s.Run(ctx)
}
