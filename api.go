package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// SearchWikipedia queries the Wikipedia REST API and decodes the response.
func searchWikipedia(ctx context.Context, client *http.Client, query string, limit int) (wikiSearchResponse, error) {
	var result wikiSearchResponse
	baseSearchURL := "https://en.wikipedia.org/w/rest.php/v1/search/page"
	u, err := url.Parse(baseSearchURL)
	if err != nil {
		return result, fmt.Errorf("parsing url: %w", err)
	}

	params := url.Values{}
	params.Add("q", query)
	params.Add("limit", strconv.Itoa(limit))
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return result, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "MyGoWikiTool/1.0 (contact: user@example.com)")

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// json.NewDecoder is generally more efficient than io.ReadAll + json.Unmarshal
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decoding response: %w", err)
	}

	return result, nil
}