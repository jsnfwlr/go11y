package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jsnfwlr/go11y"
)

// Client represents a REST API client for interacting with the widget service.
type Client struct {
	port       string
	httpClient *http.Client
}

// NewClient creates a new REST client for interacting with the widget service.
func NewClient(ctx context.Context, port string, name string) (*Client, error) {
	httpClient := go11y.HTTPClient{
		Client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: http.DefaultTransport,
		},
	}

	err := httpClient.AddLogging(ctx)
	if err != nil {
		return nil, err
	}

	err = httpClient.AddRequestID(name)
	if err != nil {
		return nil, err
	}

	tc := &Client{
		port:       port,
		httpClient: httpClient.Client,
	}

	return tc, nil
}

// FindWidget sends a GET request to the widget endpoint for the specified widget ID.
func (t *Client) FindWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	o.Info("FindWidget request sent", "url", url)

	return resp, nil
}

// UpdateWidget sends a PUT request to the widget endpoint for the specified widget ID.
func (t *Client) UpdateWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	o.Info("UpdateWidget request sent", "url", url)

	return resp, nil
}

// CreateWidget sends a POST request to the widget endpoint to create a new widget.
func (t *Client) CreateWidget(ctx context.Context) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/widget", t.port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	o.Info("CreateWidget request sent", "url", url)

	return resp, nil
}

// KillWidget sends a DELETE request to the widget endpoint for the specified widget ID.
func (t *Client) KillWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	o.Info("KillWidget request sent", "url", url)

	return resp, nil
}

// RelayFindWidget sends a GET request to the relay widget endpoint for the specified widget ID.
func (t *Client) RelayFindWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/relay/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	o.Info("RelayFindWidget request sent", "url", url)
	return resp, nil
}

// RelayUpdateWidget sends a PUT request to the relay widget endpoint for the specified widget ID.
func (t *Client) RelayUpdateWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/relay/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	o.Info("RelayUpdateWidget request sent", "url", url)
	return resp, nil
}

// RelayCreateWidget sends a POST request to the relay widget endpoint to create a new widget.
func (t *Client) RelayCreateWidget(ctx context.Context) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/relay/widget", t.port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	o.Info("RelayCreateWidget request sent", "url", url)
	return resp, nil
}

// RelayKillWidget sends a DELETE request to the relay widget endpoint for the specified widget ID.
func (t *Client) RelayKillWidget(ctx context.Context, widgetID string) (*http.Response, error) {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost:%s/relay/widget/%s", t.port, widgetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	o.Info("RelayKillWidget request sent", "url", url)
	return resp, nil
}
