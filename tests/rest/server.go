// Package rest provides a REST API server and client implementation for testing purposes.
package rest

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/gorilla/mux"

	"github.com/jsnfwlr/go11y"
)

const (
	FirstServerPort  = "9080" // Port for the first test server
	SecondServerPort = "9081" // Port for the second test server
)

// StartRESTService starts a REST API server for testing purposes. It sets up the server with the provided context,
// name, and port, and listens for termination signals on the provided channel. The server logs its activities using the
// go11y observer. The function returns an error if the server fails to start or encounters an unexpected error during
// its operation.
func StartRESTService(t *testing.T, ctx context.Context, name, port string, sigChan <-chan os.Signal, bufOut io.Writer) (fault error) {
	// Instantiate the go11y observer - a full one, not a test one
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("OTEL_URL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OTEL_SERVICE_NAME", name)
	t.Setenv("TRIM_MODULES", "")
	t.Setenv("TRIM_PATHS", "")

	cfg, err := go11y.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, cfg, bufOut, bufOut, "service_name", name)
	if err != nil {
		return fmt.Errorf("failed to initialise go11y: %v", err)
	}

	server, err := newServerInstance(ctx, port)
	if err != nil {
		return err
	}

	serverErr := make(chan error, 1)
	go func() {
		err = server.Start(ctx)
		if err != nil {
			o.Error("server closed unexpectedly", err, go11y.SeverityHighest)
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-ctx.Done(): // the context being done indicates that the server should shut down
		o.Info("Context was closed")
		o.Info("server not accepting new connections")

		if err := server.Shutdown(ctx); err != nil {
			o.Error("could not shutdown cleanly", err, go11y.SeverityHighest)
			return err
		}

		// Drain serverErr channel. Check if server returned an error after shutdown
		if err := <-serverErr; err != nil {
			o.Error("server encountered an error after shutdown", err, go11y.SeverityHighest)
			return err
		}

		return nil

	case s := <-sigChan: // received termination signal, initiate server shutdown
		o.Info("Received signal to terminate", "signal", s)
		o.Info("server not accepting new connections")

		if err := server.Shutdown(ctx); err != nil {
			o.Error("could not shutdown cleanly", err, go11y.SeverityHighest)
			return err
		}

		// Drain serverErr channel. Check if server returned an error after shutdown
		if err := <-serverErr; err != nil {
			o.Error("server encountered an error after shutdown", err, go11y.SeverityHighest)
			return err
		}

		return nil

	case err := <-serverErr:
		if err != nil {
			o.Error("server encountered an error", err, go11y.SeverityHighest)
			return err
		}
	}

	return nil
}

// ServerInstance represents an instance of the REST API server, including its address, port, router, and HTTP server.
type ServerInstance struct {
	address string
	port    string
	router  *mux.Router
	server  *http.Server
}

func newServerInstance(ctx context.Context, port string) (server *ServerInstance, fault error) {
	ctx, o, err := go11y.Get(ctx)
	if err != nil {
		return nil, err
	}

	o.Debug("initialising server")

	rtr := mux.NewRouter()

	// Get middleware for logging all requests
	requestLoggerMiddleware, err := go11y.RequestLoggerMiddlewareMux(o)
	if err != nil {
		return nil, fmt.Errorf("could not create request logger middleware: %w", err)
	}

	rh, err := newRestHandler(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("could not create rest handler: %w", err)
	}

	mainRouter := rtr.NewRoute().Subrouter()
	mainRouter.Use(go11y.SetRequestIDMiddleware, requestLoggerMiddleware)

	mainRouter.Path("/widget/{widgetID}").Handler(rh.FindWidget(ctx)).Methods(http.MethodGet)
	mainRouter.Path("/widget/{widgetID}").Handler(rh.UpdateWidget(ctx)).Methods(http.MethodPut)
	mainRouter.Path("/widget").Handler(rh.CreateWidget(ctx)).Methods(http.MethodPost)
	mainRouter.Path("/widget/{widgetID}").Handler(rh.KillWidget(ctx)).Methods(http.MethodDelete)

	mainRouter.Path("/relay/widget/{widgetID}").Handler(rh.RelayFindWidget(ctx)).Methods(http.MethodGet)
	mainRouter.Path("/relay/widget/{widgetID}").Handler(rh.RelayUpdateWidget(ctx)).Methods(http.MethodPut)
	mainRouter.Path("/relay/widget").Handler(rh.RelayCreateWidget(ctx)).Methods(http.MethodPost)
	mainRouter.Path("/relay/widget/{widgetID}").Handler(rh.RelayKillWidget(ctx)).Methods(http.MethodDelete)

	s := &ServerInstance{
		port:    port,
		address: "0.0.0.0",
		router:  rtr,
		server: &http.Server{
			Addr:    net.JoinHostPort("0.0.0.0", port),
			Handler: rtr,
		},
	}

	return s, nil
}

// Start starts the REST API HTTP server.
func (s *ServerInstance) Start(ctx context.Context) error {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return err
	}

	o.Debug("starting server", "address", s.address, "port", s.port)
	err = s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		o.Error("server error", err, go11y.SeverityHighest)
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the REST API HTTP server.
func (s *ServerInstance) Shutdown(ctx context.Context) error {
	_, o, err := go11y.Get(ctx)
	if err != nil {
		go11y.Error("could not get observer from context", err, go11y.SeverityHighest)
		return err
	}

	o.Debug("shutting down server")
	return s.server.Shutdown(ctx)
}

// func healthcheckFunc(ctx context.Context) (passed bool, body []byte) {
// 	return true, []byte{}
// }

// func maskPath(path string) (maskedPath string) {
// 	// Remove the unique identifiers from the path
// 	rEx := regexp.MustCompile(`/(widget)_[0-9A-Z]+(.*)`)
// 	path = rEx.ReplaceAllString(path, "/${1}_id${2}")

// 	return path
// }
