package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/davidseybold/unifi-dns-sync/dnsprovider/technitium"
	"github.com/davidseybold/unifi-dns-sync/syncer"
	"github.com/davidseybold/unifi-dns-sync/unifi"
)

type config struct {
	UnifiAPIURL string `env:"UNIFI_API_URL"`
	UnifiAPIKey string `env:"UNIFI_API_KEY"`
	UnifiSiteID string `env:"UNIFI_SITE_ID"`

	SyncZone string `env:"SYNC_ZONE"`
	StateDir string `env:"STATE_DIR" envDefault:"/var/lib/unifi-sync"`

	DNSProvider string `env:"DNS_PROVIDER"`
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

	if c.DNSProvider == "" {
		return errors.New("DNS_PROVIDER is required")
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

func getProvider(name string) (syncer.DNSProvider, error) {
	switch name {
	case "technitium":
		p, err := technitium.New()
		return p, err
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
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

	logger.Info("Starting sync", "provider", config.DNSProvider)

	unifiClient := unifi.NewClient(config.UnifiAPIURL, config.UnifiAPIKey, config.UnifiSiteID)

	provider, err := getProvider(strings.ToLower(config.DNSProvider))
	if err != nil {
		return fmt.Errorf("error creating provider: %v", err)
	}

	syncerCfg := syncer.Config{
		SyncZone:       config.SyncZone,
		StateDir:       config.StateDir,
		ClientWaitTime: time.Hour,
	}

	s := syncer.New(unifiClient, provider, syncerCfg, logger)

	result, err := s.Run(ctx)
	if err != nil {
		return fmt.Errorf("error running sync: %v", err)
	}

	logger.Info("Sync completed", "recordsUpserted", result.AddSuccess, "recordsUpsertedFailed", result.AddFailed, "recordsDeleted", result.DeleteSuccess, "recordsDeletedFailed", result.DeleteFailed)

	return nil
}
