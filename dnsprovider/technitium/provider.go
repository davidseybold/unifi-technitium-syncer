package technitium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/davidseybold/unifi-dns-sync/dnsprovider"
)

var ()

// Client holds the configuration for the API
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type config struct {
	BaseURL  string `env:"TECHNITIUM_API_URL"`
	APIToken string `env:"TECHNITIUM_API_TOKEN"`
}

func (c *config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("TECHNITIUM_API_URL is required for technitium provider")
	}

	if c.APIToken == "" {
		return errors.New("TECHNITIUM_API_TOKEN is required for technitium provider")
	}

	return nil
}

func loadConfig() (config, error) {
	var c config
	err := env.Parse(&c)
	return c, err
}

func New() (*Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		BaseURL: cfg.BaseURL,
		Token:   cfg.APIToken,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) listZones(ctx context.Context) ([]Zone, error) {

	response, err := doRequest[ListZonesResponse](ctx, c, "/api/zones/list", url.Values{})
	if err != nil {
		return nil, err
	}

	return response.Zones, nil
}

// GetZone fetches details for a specific zone
func (c *Client) GetZone(ctx context.Context, zoneName string) (dnsprovider.Zone, error) {
	zones, err := c.listZones(ctx)
	if err != nil {
		return dnsprovider.Zone{}, err
	}

	for i := range zones {
		if zones[i].Name == zoneName {
			return zones[i].ToDNSProviderZone(), nil
		}
	}

	return dnsprovider.Zone{}, dnsprovider.ErrZoneNotFound
}

func (c *Client) UpsertRecord(ctx context.Context, zone string, record dnsprovider.Record) error {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", record.Name)
	params.Add("type", record.Type)
	params.Add("ttl", fmt.Sprintf("%d", record.TTL))
	params.Add("comments", record.Comments)
	params.Add("ptr", "true")
	params.Add("createPtrZone", "true")
	params.Add("overwrite", "true")
	params.Add("ipAddress", record.IPAddress)

	_, err := doRequest[struct{}](ctx, c, "/api/zones/records/add", params)

	return err
}

// ListRecords returns all records for a zone
func (c *Client) ListRecords(ctx context.Context, zone string) ([]dnsprovider.Record, error) {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", zone)
	params.Add("listZone", "true")

	records, err := doRequest[ListRecordsResponse](ctx, c, "/api/zones/records/get", params)
	if err != nil {
		return nil, err
	}

	var result []dnsprovider.Record
	for i := range records.Records {
		result = append(result, records.Records[i].ToDNSProviderRecord())
	}

	return result, nil
}

// // DeleteRecord removes a record from a zone
func (c *Client) DeleteRecord(ctx context.Context, zone string, record dnsprovider.Record) error {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", record.Name)
	params.Add("type", record.Type)
	params.Add("ipAddress", record.IPAddress)

	_, err := doRequest[struct{}](ctx, c, "/api/zones/records/delete", params)

	return err
}

func doRequest[T any](ctx context.Context, c *Client, path string, params url.Values) (*T, error) {
	params.Add("token", c.Token)

	fullURL := fmt.Sprintf("%s%s?%s", c.BaseURL, path, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error: %s (status: %d)", resp.Status, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[T]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base response: %v", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("api error: %s (status: %s)", apiResp.ErrorMessage, resp.Status)
	}

	return &apiResp.Response, nil
}
