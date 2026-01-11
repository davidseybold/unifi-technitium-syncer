package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/davidseybold/unifi-technitium-sync/syncer"
	"github.com/davidseybold/unifi-technitium-sync/technitium"
	"github.com/davidseybold/unifi-technitium-sync/unifi"
)

type config struct {
	UnifiAPIURL        string `env:"UNIFI_API_URL"`
	UnifiAPIKey        string `env:"UNIFI_API_KEY"`
	UnifiSiteID        string `env:"UNIFI_SITE_ID"`
	TechnitiumAPIURL   string `env:"TECHNITIUM_API_URL"`
	TechnitiumAPIToken string `env:"TECHNITIUM_API_TOKEN"`
	SyncZone           string `env:"SYNC_ZONE"`
}

func (c *config) String() string {
	return fmt.Sprintf("UnifiAPIURL: %s, TechnitiumAPIURL: %s, SyncZone: %s", c.UnifiAPIURL, c.TechnitiumAPIURL, c.SyncZone)
}

func (c *config) Validate() error {
	if c.UnifiAPIURL == "" {
		return errors.New("UNIFI_API_URL is required")
	}

	if c.UnifiAPIKey == "" {
		return errors.New("UNIFI_API_KEY is required")
	}

	if c.UnifiSiteID == "" {
		return errors.New("UNIFI_SITE_ID is required")
	}

	if c.TechnitiumAPIURL == "" {
		return errors.New("TECHNITIUM_API_URL is required")
	}

	if c.TechnitiumAPIToken == "" {
		return errors.New("TECHNITIUM_API_TOKEN is required")
	}

	if c.SyncZone == "" {
		return errors.New("SYNC_ZONE is required")
	}

	return nil
}

func LoadConfig() (config, error) {
	_ = godotenv.Load()

	var c config
	err := env.Parse(&c)
	return c, err
}

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
}

func run(ctx context.Context) error {

	runId := uuid.New().String()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger = logger.With("runId", runId)

	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("error validating config: %v", err)
	}

	logger.Info("Starting sync", "config", config.String())

	unifiClient := unifi.NewClient(config.UnifiAPIURL, config.UnifiAPIKey, config.UnifiSiteID)
	technitiumClient := technitium.NewClient(config.TechnitiumAPIURL, config.TechnitiumAPIToken)

	s := syncer.New(unifiClient, technitiumClient, config.SyncZone, logger)

	return s.Run(ctx)
}
