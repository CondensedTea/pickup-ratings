package tf2pickup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"log/slog"
)

type Client struct {
	tr http.RoundTripper

	apiHost  string
	pageSize int
}

func NewClient(pickupURL string, pageSize int, tr http.RoundTripper) *Client {
	return &Client{
		tr:       tr,
		pageSize: pageSize,
		apiHost:  "api." + pickupURL,
	}
}

func (c *Client) LoadNewGames(ctx context.Context, startingOffset, limit int) ([]Result, error) {
	var totalResults []Result

	for i := startingOffset; ; i += c.pageSize {
		results, resultCount, err := c.loadResultsPage(ctx, c.pageSize, i)
		if err != nil {
			return nil, err
		}

		totalResults = append(totalResults, results...)

		slog.Info("loaded results page",
			"offset", c.pageSize,
			"last_id", i,
			"total_results", len(totalResults),
		)

		isLastGamePlayed := results[len(results)-1].Number == resultCount

		if len(totalResults) >= limit || isLastGamePlayed {
			return totalResults, nil
		}
	}
}

func (c *Client) loadResultsPage(ctx context.Context, limit, offset int) ([]Result, int64, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     c.apiHost,
		Path:     "/games",
		RawQuery: fmt.Sprintf("limit=%d&offset=%d&sort=launchedAt", limit, offset),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("preparing http request: %w", err)
	}

	resp, err := c.tr.RoundTrip(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
	}

	type results struct {
		Results   []Result `json:"results"`
		ItemCount int64    `json:"itemCount"`
	}

	var v results
	if err = json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, 0, err
	}

	return v.Results, v.ItemCount, nil
}
