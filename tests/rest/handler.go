package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/jsnfwlr/go11y"
)

type restHandler struct {
	testClient *Client
}

func newRestHandler(ctx context.Context, serverPort string) (rh *restHandler, fault error) {
	clientPort := FirstServerPort
	if serverPort == FirstServerPort {
		clientPort = SecondServerPort
	}

	httpClient := go11y.HTTPClient{
		Client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: http.DefaultTransport,
		},
	}

	name := "unknown-client"
	switch serverPort {
	case FirstServerPort:
		name = "first-server-client"
	case SecondServerPort:
		name = "second-server-client"
	}

	err := httpClient.AddRequestID(name)
	if err != nil {
		return nil, err
	}

	err = httpClient.AddLogging(ctx)
	if err != nil {
		return nil, err
	}

	tc := &Client{
		port:       clientPort,
		httpClient: httpClient.Client,
	}

	return &restHandler{
		testClient: tc,
	}, nil
}

func (rh *restHandler) FindWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling FindWidget request")

		w.Header().Add("Content-Type", "application/json")
		widgetID := mux.Vars(r)["widgetID"]
		resp := map[string]any{
			"message": fmt.Sprintf("Found widget with ID %s", widgetID),
			"widgetID": map[string]any{
				"id": widgetID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (rh *restHandler) UpdateWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling UpdateWidget request")
		w.Header().Add("Content-Type", "application/json")
		widgetID := mux.Vars(r)["widgetID"]
		resp := map[string]any{
			"message": fmt.Sprintf("Updated widget with ID %s", widgetID),
			"widgetID": map[string]any{
				"id": widgetID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (rh *restHandler) CreateWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling CreateWidget request")

		w.Header().Add("Content-Type", "application/json")
		widgetID := "widget_123456789ABC"
		resp := map[string]any{
			"message": fmt.Sprintf("Created widget with ID %s", widgetID),
			"widgetID": map[string]any{
				"id": widgetID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (rh *restHandler) KillWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling KillWidget request")

		w.Header().Add("Content-Type", "application/json")
		widgetID := mux.Vars(r)["widgetID"]
		resp := map[string]any{
			"message": fmt.Sprintf("Killed widget with ID %s", widgetID),
			"widgetID": map[string]any{
				"id": widgetID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (rh *restHandler) RelayFindWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling RelayFindWidget request")

		widgetID := mux.Vars(r)["widgetID"]
		resp, err := rh.testClient.FindWidget(ctx, widgetID)
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		w.Header().Add("Content-Type", "application/json")
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
		}
		_, _ = io.Copy(w, resp.Body)
	}
}

func (rh *restHandler) RelayUpdateWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling RelayUpdateWidget request")
		widgetID := mux.Vars(r)["widgetID"]
		resp, err := rh.testClient.UpdateWidget(ctx, widgetID)
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		w.Header().Add("Content-Type", "application/json")
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
		}
		_, _ = io.Copy(w, resp.Body)
	}
}

func (rh *restHandler) RelayCreateWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling RelayCreateWidget request")

		resp, err := rh.testClient.CreateWidget(ctx)
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		w.Header().Add("Content-Type", "application/json")
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
		}
		_, _ = io.Copy(w, resp.Body)
	}
}

func (rh *restHandler) RelayKillWidget(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, o, err := go11y.Get(r.Context())
		if err != nil {
			go11y.Error("no go11y observer in request context", err, go11y.SeverityHigh)
			http.Error(w, "no go11y observer in request context", http.StatusInternalServerError)
			return
		}

		o.Info("handling RelayKillWidget request")

		widgetID := mux.Vars(r)["widgetID"]
		resp, err := rh.testClient.KillWidget(ctx, widgetID)
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		w.Header().Add("Content-Type", "application/json")
		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
		}
		_, _ = io.Copy(w, resp.Body)
	}
}
