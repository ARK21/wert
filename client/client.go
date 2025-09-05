package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ARK21/wert/domain"
)

type APIClient struct {
	*http.Client
	baseUrl *url.URL
	apiKey  string
}

func NewAPIClient(baseUrl, apiKey string) (*APIClient, error) {
	parsed, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create api client: %w", err)
	}

	return &APIClient{
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseUrl: parsed,
		apiKey:  apiKey,
	}, nil
}

func (c *APIClient) Exchange(ctx context.Context, ex domain.Exchange) (float64, error) {
	req, err := c.cmcReq(ctx, ex)
	if err != nil {
		return 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, fmt.Errorf("could not get data from CMC: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("could not read data from CMC: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("CMC unexpected status code: %s", resp.Status)
	}

	res, err := c.cmcRes(body)
	if err != nil {
		return 0, err
	}

	d, ok := res.Data[ex.From]
	if !ok {
		return 0, fmt.Errorf("missing data for %q", ex.From)
	}

	qt, ok := d.Quote[ex.To]
	if !ok {
		return 0, fmt.Errorf("missing quote for %q", ex.To)
	}

	return qt.Price, nil
}

func (c *APIClient) cmcRes(body []byte) (*cmcRes, error) {
	var res cmcRes
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("could not decode data from CMC: %w", err)
	}

	return &res, nil
}

func (c *APIClient) cmcReq(ctx context.Context, ex domain.Exchange) (*http.Request, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v2/tools/price-conversion")
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-CMC_PRO_API_KEY", c.apiKey)

	q := url.Values{}
	q.Add("amount", fmt.Sprintf("%.2f", ex.Amount))
	q.Add("symbol", ex.From)
	q.Add("convert", ex.To)
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func (c *APIClient) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	fullURL := c.baseUrl.ResolveReference(rel)
	return http.NewRequestWithContext(ctx, method, fullURL.String(), nil)
}

type cmcRes struct {
	Data map[string]data `json:"data"`
}

type data struct {
	Quote map[string]quote `json:"quote"`
}

type quote struct {
	Price float64 `json:"price"`
}
