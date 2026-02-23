package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultSavingsAPI = "https://services.publix.com/api/v4/savings"
	defaultStoreAPI   = "https://services.publix.com/api/v1/storelocation"
	userAgent         = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
)

// Client is an HTTP client for the Publix API.
type Client struct {
	httpClient *http.Client
	savingsURL string
	storeURL   string
}

// NewClient creates a new Publix API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		savingsURL: defaultSavingsAPI,
		storeURL:   defaultStoreAPI,
	}
}

// NewClientWithBaseURLs creates a client with custom base URLs (for testing).
func NewClientWithBaseURLs(savingsURL, storeURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		savingsURL: savingsURL,
		storeURL:   storeURL,
	}
}

func (c *Client) get(ctx context.Context, reqURL, storeNumber string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if storeNumber != "" {
		req.Header.Set("PublixStore", storeNumber)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, reqURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return body, nil
}

// FetchStores finds Publix stores near the given zip code.
func (c *Client) FetchStores(ctx context.Context, zipCode string, count int) ([]Store, error) {
	params := url.Values{
		"types":                    {"R,G,H,N,S"},
		"option":                   {""},
		"count":                    {fmt.Sprintf("%d", count)},
		"includeOpenAndCloseDates": {"true"},
		"zipCode":                  {zipCode},
	}

	body, err := c.get(ctx, c.storeURL+"?"+params.Encode(), "")
	if err != nil {
		return nil, fmt.Errorf("fetching stores: %w", err)
	}

	var resp StoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding stores: %w", err)
	}
	return resp.Stores, nil
}

// FetchSavings fetches all weekly ad savings for the given store.
func (c *Client) FetchSavings(ctx context.Context, storeNumber string) (*SavingsResponse, error) {
	params := url.Values{
		"page":                    {"1"},
		"pageSize":                {"0"},
		"includePersonalizedDeals": {"false"},
		"languageID":              {"1"},
		"isWeb":                   {"true"},
		"getSavingType":           {"WeeklyAd"},
	}

	body, err := c.get(ctx, c.savingsURL+"?"+params.Encode(), storeNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching savings: %w", err)
	}

	var resp SavingsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding savings: %w", err)
	}
	return &resp, nil
}

// StoreNumber returns the numeric portion of a store key (strips leading zeros).
func StoreNumber(key string) string {
	return strings.TrimLeft(key, "0")
}
