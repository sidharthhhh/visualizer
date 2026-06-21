package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

type Sample struct {
	Metric string
	Value  float64
	Labels map[string]string
	Time   time.Time
}

func (c *Client) Write(ctx context.Context, samples []Sample) error {
	var buf bytes.Buffer

	for _, s := range samples {
		labels := ""
		for k, v := range s.Labels {
			if labels != "" {
				labels += ","
			}
			labels += fmt.Sprintf(`%s="%s"`, k, v)
		}

		line := fmt.Sprintf("%s{%s} %f %d\n",
			s.Metric,
			labels,
			s.Value,
			s.Time.UnixMilli(),
		)
		buf.WriteString(line)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/import/prometheus", &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

type QueryResult struct {
	Metric map[string]string
	Value  []interface{}
}

func (c *Client) QueryInstant(ctx context.Context, query string) ([]QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/query?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Result []QueryResult `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.Data.Result, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/query_range?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Result []QueryResult `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.Data.Result, nil
}
