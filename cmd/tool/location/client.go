package location

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"resty.dev/v3"
)

// Client is a lightweight wrapper around the Geonames.org REST API.
type Client struct {
	// Base URL for the Geonames API (default: http://api.geonames.org)
	baseURL string
	// Username is required by Geonames for all requests.
	username string
	// Resty client used for all requests.
	httpClient *resty.Client
}

// NewClient creates a new Client with the given username.
// If httpClient is nil, a default Resty client with a 10‑second timeout is used.
func NewClient(username string, httpClient *resty.Client) *Client {
	if httpClient == nil {
		httpClient = resty.New().
			SetTimeout(10 * time.Second).
			SetRetryCount(0) // no automatic retries – we handle errors explicitly
	}
	return &Client{
		baseURL:    "http://api.geonames.org",
		username:   username,
		httpClient: httpClient,
	}
}

// SearchParams holds optional parameters for the search endpoint.
type SearchParams struct {
	// Name of the place to search for (required).
	Q string
	// Max number of results to return.
	MaxRows int
}

// SearchResult represents a single search hit.
type SearchResult struct {
	Name        string `json:"name"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lng"`
	FeatureCode string `json:"fcode"`
}

// SearchResponse is the JSON payload returned by the search endpoint.
type SearchResponse struct {
	TotalResultsCount int            `json:"totalResultsCount"`
	Geonames          []SearchResult `json:"geonames"`
}

// Search queries the /searchJSON endpoint for places matching the given parameters.
func (c *Client) Search(ctx context.Context, params SearchParams) (*SearchResponse, error) {
	if params.Q == "" {
		return nil, fmt.Errorf("search query (Q) is required")
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"q":        params.Q,
			"username": c.username,
		}).
		SetQueryParam("maxRows", fmt.Sprintf("%d", params.MaxRows)).
		Get(c.baseURL + "/searchJSON")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("geonames search returned status %d", resp.StatusCode())
	}

	var sr SearchResponse
	if err := json.Unmarshal(resp.Bytes(), &sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// GetPlaceParams holds optional parameters for the getJSON endpoint.
type GetPlaceParams struct {
	// Geoname ID of the place to retrieve (required).
	ID string
}

// GetPlaceResponse represents the detailed information for a single place.
type GetPlaceResponse struct {
	Name        string `json:"name"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lng"`
	FeatureCode string `json:"fcode"`
	// Additional fields can be added as needed.
}

// GetPlace retrieves detailed information for a place by its Geoname ID.
func (c *Client) GetPlace(ctx context.Context, params GetPlaceParams) (*GetPlaceResponse, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("geoname ID is required")
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"geonameId": params.ID,
			"username":  c.username,
		}).
		Get(c.baseURL + "/getJSON")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("geonames get returned status %d", resp.StatusCode())
	}

	var gp GetPlaceResponse
	if err := json.Unmarshal(resp.Bytes(), &gp); err != nil {
		return nil, err
	}
	return &gp, nil
}
