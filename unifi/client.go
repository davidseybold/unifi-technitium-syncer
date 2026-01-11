package unifi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	SiteID     string
}

func NewClient(baseURL, apiKey string, siteID string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		SiteID:  siteID,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

type ListClientsResponse struct {
	Offset     int             `json:"offset"`
	Limit      int             `json:"limit"`
	Count      int             `json:"count"`
	TotalCount int             `json:"totalCount"`
	Data       []NetworkClient `json:"data"`
}

type NetworkClient struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress"`
	MacAddress string `json:"macAddress"`
}

func (c *Client) ListClients(ctx context.Context) ([]NetworkClient, error) {
	allClients := []NetworkClient{}
	limit := 50
	offset := 0

	for {
		endpoint := fmt.Sprintf("/proxy/network/integration/v1/sites/%s/clients", c.SiteID)
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", limit))
		params.Set("offset", fmt.Sprintf("%d", offset))

		var response ListClientsResponse
		if err := c.doRequest(ctx, "GET", endpoint, params, &response); err != nil {
			return nil, fmt.Errorf("request failed at offset %d: %w", offset, err)
		}

		allClients = append(allClients, response.Data...)

		if len(allClients) >= response.TotalCount || len(response.Data) == 0 {
			break
		}

		offset += limit
	}

	return allClients, nil
}

func (c *Client) doRequest(ctx context.Context, method string, path string, params url.Values, out any) error {
	fullURL := fmt.Sprintf("%s%s", c.BaseURL, path)

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("X-API-KEY", c.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if out != nil {
		return json.Unmarshal(body, out)
	}

	return nil
}
