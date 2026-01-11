package unifi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	var response ListClientsResponse
	if err := c.doRequest(ctx, "GET", "/proxy/network/integration/v1/sites/"+c.SiteID+"/clients", &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *Client) doRequest(ctx context.Context, method string, path string, out any) error {
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
