package civitai

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

const BaseURL = "https://civitai.com/api/v1"

// Client is a wrapper around the resty client for Civitai API calls.
type Client struct {
	http *resty.Client
	cfg  *config.Config
}

// NewClient initializes a new Civitai API client.
func NewClient(cfg *config.Config) *Client {
	r := resty.New()
	r.SetBaseURL(BaseURL)
	r.SetTimeout(time.Duration(cfg.TimeoutSeconds) * time.Second)
	r.SetRetryCount(cfg.RetryCount)
	r.SetRetryWaitTime(2 * time.Second)
	r.SetRetryMaxWaitTime(10 * time.Second)

	// Inject Authorization header if API Key exists
	if cfg.APIKey != "" {
		r.SetAuthToken(cfg.APIKey)
	}

	return &Client{
		http: r,
		cfg:  cfg,
	}
}

// SearchModels queries the models endpoint.
func (c *Client) SearchModels(ctx context.Context, opts SearchModelsOptions) (*ListResponse[Model], error) {
	if opts.Limit <= 0 {
		opts.Limit = c.cfg.DefaultSearchLimit
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	req := c.http.R().SetContext(ctx).
		SetQueryParam("limit", strconv.Itoa(opts.Limit)).
		SetResult(&ListResponse[Model]{}).
		SetError(&map[string]interface{}{})

	if opts.Query != "" {
		req.SetQueryParam("query", opts.Query)
	} else {
		req.SetQueryParam("page", strconv.Itoa(opts.Page))
	}

	if len(opts.Types) > 0 {
		for _, t := range opts.Types {
			req.SetQueryParam("types", t)
		}
	}
	if opts.Sort != "" {
		req.SetQueryParam("sort", opts.Sort)
	}
	if opts.Period != "" {
		req.SetQueryParam("period", opts.Period)
	}
	if opts.Rating > 0 {
		req.SetQueryParam("rating", strconv.Itoa(opts.Rating))
	}
	req.SetQueryParam("nsfw", strconv.FormatBool(opts.NSFW))

	log.Debug().Msgf("Searching models with query: %s, limit: %d", opts.Query, opts.Limit)
	resp, err := req.Get("/models")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", resp.String())
	}

	result, ok := resp.Result().(*ListResponse[Model])
	if !ok {
		return nil, fmt.Errorf("failed to parse response")
	}

	return result, nil
}

// GetModel retrieves a specific model by ID.
func (c *Client) GetModel(ctx context.Context, id int) (*Model, error) {
	req := c.http.R().SetContext(ctx).
		SetResult(&Model{}).
		SetError(&map[string]interface{}{})

	path := fmt.Sprintf("/models/%d", id)
	resp, err := req.Get(path)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("model %d not found", id)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", resp.String())
	}

	result, ok := resp.Result().(*Model)
	if !ok {
		return nil, fmt.Errorf("failed to parse response")
	}

	return result, nil
}

// GetImages retrieves a list of images, optionally filtered by modelId.
func (c *Client) GetImages(ctx context.Context, modelID int, limit int, page int) (*ListResponse[Image], error) {
	if limit <= 0 {
		limit = 100 // API default
	}
	if page <= 0 {
		page = 1
	}

	req := c.http.R().SetContext(ctx).
		SetQueryParam("limit", strconv.Itoa(limit)).
		SetQueryParam("page", strconv.Itoa(page)).
		SetResult(&ListResponse[Image]{}).
		SetError(&map[string]interface{}{})

	if modelID > 0 {
		req.SetQueryParam("modelId", strconv.Itoa(modelID))
	}

	resp, err := req.Get("/images")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", resp.String())
	}

	result, ok := resp.Result().(*ListResponse[Image])
	if !ok {
		return nil, fmt.Errorf("failed to parse response")
	}

	return result, nil
}
