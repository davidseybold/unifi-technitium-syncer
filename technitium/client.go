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
)

var (
	ErrZoneNotFound = errors.New("zone not found")
)

// Client holds the configuration for the API
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// BaseResponse covers the standard status fields returned by all API calls
type APIResponse[T any] struct {
	Response     T      `json:"response"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type ListZonesResponse struct {
	Zones []DNSZone `json:"zones"`
}

type DNSZone struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ListRecordsResponse struct {
	Records []Record `json:"records"`
}

type Record struct {
	Type     string `json:"type"`
	TTL      int    `json:"ttl"`
	Name     string `json:"name"`
	Comments string `json:"comments"`
	RData    RData  `json:"rData"`
}

type RData struct {
	IPAddress *string `json:"ipAddress"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateZone creates a new primary zone
func (c *Client) CreateZone(ctx context.Context, zoneName string) error {
	params := url.Values{}
	params.Add("zone", zoneName)
	params.Add("type", "Primary") // Defaulting to Primary for this example

	_, err := doRequest[struct{}](ctx, c, "/api/zones/create", params)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ListZones(ctx context.Context) ([]DNSZone, error) {

	response, err := doRequest[ListZonesResponse](ctx, c, "/api/zones/list", url.Values{})
	if err != nil {
		return nil, err
	}

	return response.Zones, nil
}

// GetZone fetches details for a specific zone
func (c *Client) GetZone(ctx context.Context, zoneName string) (*DNSZone, error) {
	zones, err := c.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	for i := range zones {
		if zones[i].Name == zoneName {
			return &zones[i], nil
		}
	}

	return nil, ErrZoneNotFound
}

// // AddRecord adds a DNS record to a zone
func (c *Client) AddRecord(ctx context.Context, zone, domain, ipAddress string, ttl int, comments string) error {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", domain)
	params.Add("type", "A")
	params.Add("ttl", fmt.Sprintf("%d", ttl))
	params.Add("comments", comments)
	params.Add("ptr", "true")
	params.Add("createPtrZone", "true")
	params.Add("overwrite", "true")
	params.Add("ipAddress", ipAddress)

	_, err := doRequest[struct{}](ctx, c, "/api/zones/records/add", params)

	return err
}

// ListRecords returns all records for a zone
func (c *Client) ListRecords(ctx context.Context, zone string) ([]Record, error) {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", zone)
	params.Add("listZone", "true")

	records, err := doRequest[ListRecordsResponse](ctx, c, "/api/zones/records/get", params)
	if err != nil {
		return nil, err
	}

	return records.Records, nil
}

// // DeleteRecord removes a record from a zone
func (c *Client) DeleteRecord(ctx context.Context, zone, domain, recordType, value string) error {
	params := url.Values{}
	params.Add("zone", zone)
	params.Add("domain", domain)
	params.Add("type", recordType)

	if recordType == "A" {
		params.Add("ipAddress", value)
	}

	_, err := doRequest[struct{}](ctx, c, "/api/zones/records/delete", params)

	return err
}

func doRequest[T any](ctx context.Context, c *Client, path string, params url.Values) (*T, error) {
	params.Add("token", c.Token)

	fullURL := fmt.Sprintf("%s%s?%s", c.BaseURL, path, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

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
