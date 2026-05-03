package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

const (
	defaultGeoBaseURL     = "https://api.openweathermap.org/geo/1.0/direct"
	defaultWeatherBaseURL = "https://api.openweathermap.org/data/2.5/weather"
)

type Client struct {
	HTTPClient     *http.Client
	GeoBaseURL     string
	WeatherBaseURL string
}

func New() *Client {
	return &Client{
		HTTPClient:     &http.Client{Timeout: 10 * time.Second},
		GeoBaseURL:     defaultGeoBaseURL,
		WeatherBaseURL: defaultWeatherBaseURL,
	}
}

func (c *Client) ResolveLocation(ctx context.Context, apiKey, query string) (pool.WeatherLocation, error) {
	apiKey = strings.TrimSpace(apiKey)
	query = strings.TrimSpace(query)
	if apiKey == "" {
		return pool.WeatherLocation{}, fmt.Errorf("openweathermap api key is required")
	}
	if query == "" {
		return pool.WeatherLocation{}, fmt.Errorf("weather location is required")
	}
	endpoint, err := url.Parse(c.baseURL(c.GeoBaseURL, defaultGeoBaseURL))
	if err != nil {
		return pool.WeatherLocation{}, err
	}
	values := endpoint.Query()
	values.Set("q", query)
	values.Set("limit", "1")
	values.Set("appid", apiKey)
	endpoint.RawQuery = values.Encode()

	var response []struct {
		Name    string            `json:"name"`
		Local   map[string]string `json:"local_names"`
		Lat     float64           `json:"lat"`
		Lon     float64           `json:"lon"`
		Country string            `json:"country"`
		State   string            `json:"state"`
	}
	if err := c.getJSON(ctx, endpoint.String(), &response); err != nil {
		return pool.WeatherLocation{}, err
	}
	if len(response) == 0 {
		return pool.WeatherLocation{}, fmt.Errorf("weather location %q was not found", query)
	}
	name := response[0].Name
	if english := response[0].Local["en"]; english != "" {
		name = english
	}
	return pool.WeatherLocation{
		Query:   query,
		Name:    name,
		Country: response[0].Country,
		State:   response[0].State,
		Lat:     response[0].Lat,
		Lon:     response[0].Lon,
	}, nil
}

func (c *Client) CurrentWeather(ctx context.Context, apiKey string, location pool.WeatherLocation) (json.RawMessage, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("openweathermap api key is required")
	}
	if location.Lat == 0 && location.Lon == 0 {
		return nil, fmt.Errorf("resolved weather location is required")
	}
	endpoint, err := url.Parse(c.baseURL(c.WeatherBaseURL, defaultWeatherBaseURL))
	if err != nil {
		return nil, err
	}
	values := endpoint.Query()
	values.Set("lat", fmt.Sprintf("%.6f", location.Lat))
	values.Set("lon", fmt.Sprintf("%.6f", location.Lon))
	values.Set("appid", apiKey)
	values.Set("units", "metric")
	endpoint.RawQuery = values.Encode()

	return c.getRaw(ctx, endpoint.String())
}

func (c *Client) getJSON(ctx context.Context, url string, target any) error {
	body, err := c.getRaw(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func (c *Client) getRaw(ctx context.Context, url string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openweathermap request failed: %s", resp.Status)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("openweathermap returned invalid json")
	}
	return json.RawMessage(body), nil
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) baseURL(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
