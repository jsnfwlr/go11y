package go11y_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
	"uuid"

	"github.com/jsnfwlr/go11y"
	"github.com/jsnfwlr/go11y/tests/rest"
)

// This is intended to be a full comprehensive live test for the go11y package, covering all the (non-deprecated)
// transport and middleware functionality as they would be used in a real application.
func TestGo11y(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Instantiate the go11y observer - a full one, not a test one
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("OTEL_URL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OTEL_SERVICE_NAME", "go11y-client")
	t.Setenv("TRIM_MODULES", "")
	t.Setenv("TRIM_PATHS", fmt.Sprintf("/usr/local/go/, %s", dir))

	cfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	bufOut := new(bytes.Buffer)

	ctx, _, err := go11y.Initialise(t.Context(), cfg, bufOut, bufOut, "client_name", "go11y_test")
	if err != nil {
		t.Fatalf("Failed to initialise go11y: %v", err)
	}

	firstCtx, firstCancel := context.WithCancel(context.Background())
	defer firstCancel()

	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err = rest.StartRESTService(t, firstCtx, "first-service", rest.FirstServerPort, sigChan, bufOut)
		if err != nil {
			t.Errorf("Failed to start first REST service: %v", err)
		}
	}()

	go func() {
		err = rest.StartRESTService(t, secondCtx, "second-service", rest.SecondServerPort, sigChan, bufOut)
		if err != nil {
			t.Errorf("Failed to start second REST service: %v", err)
		}
	}()

	client1, err := rest.NewClient(ctx, rest.FirstServerPort, "test-client-1")
	if err != nil {
		t.Fatalf("Failed to create first client: %v", err)
	}

	client2, err := rest.NewClient(ctx, rest.SecondServerPort, "test-client-2")
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}

	testCases := []struct {
		name               string
		client             *rest.Client
		route              string
		requestID          uuid.UUID
		request            string
		expectedLogs       []string
		allowedErrorRegexp []*regexp.Regexp
	}{
		{
			name:      "client1 relay preset create",
			client:    client1,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:06:29.753922+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"POST","request_url":"http://localhost:9080/relay/widget","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.764536+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-1","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.764584+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayCreateWidget.func1","file":"/tests/rest/handler.go","line":218},"msg":"handling RelayCreateWidget request","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-1","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.764656+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":104},"msg":"outbound call - request","service_name":"first-service","request_headers":{},"request_method":"POST","request_url":"http://localhost:9081/widget","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.766543+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62645","user_agent":"first-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.766613+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:62645","user_agent":"first-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.766656+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62645","user_agent":"first-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.766962+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":104},"msg":"outbound call - response","service_name":"first-service","call_duration":2182125,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:06:29 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:06:29.767001+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-1","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9081/widget"}`,
				`{"time":"2026-09-24T14:06:29.76706+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-1","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.767212+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":12775625,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:06:29 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:06:29.767241+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayCreateWidget","file":"/tests/rest/client.go","line":197},"msg":"RelayCreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/relay/widget"}`,
			},
		},

		{
			name:      "client1 relay preset find",
			client:    client1,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:21.140373+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"GET","request_url":"http://localhost:9080/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.14051+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-1","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140525+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayFindWidget.func1","file":"/tests/rest/handler.go","line":161},"msg":"handling RelayFindWidget request","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-1","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140538+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":57},"msg":"outbound call - request","service_name":"first-service","request_headers":{},"request_method":"GET","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140613+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62797","user_agent":"first-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140629+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:62797","user_agent":"first-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140636+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62797","user_agent":"first-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140706+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":57},"msg":"outbound call - response","service_name":"first-service","call_duration":145292,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:21 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:09:21.140713+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-1","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9081/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:21.140754+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-1","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140842+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":428959,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:21 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:09:21.14085+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayFindWidget","file":"/tests/rest/client.go","line":154},"msg":"RelayFindWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/relay/widget/widget_1000"}`,
			},
		},
		{
			name:      "client1 relay preset update",
			client:    client1,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:55.869954+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"PUT","request_url":"http://localhost:9080/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870258+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870272+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayUpdateWidget.func1","file":"/tests/rest/handler.go","line":190},"msg":"handling RelayUpdateWidget request","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870293+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":79},"msg":"outbound call - request","service_name":"first-service","request_headers":{},"request_method":"PUT","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870424+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870434+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870451+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870583+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":79},"msg":"outbound call - response","service_name":"first-service","call_duration":244625,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:09:55.870603+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9081/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:55.870626+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870772+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":727375,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:09:55.870796+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayUpdateWidget","file":"/tests/rest/client.go","line":174},"msg":"RelayUpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/relay/widget/widget_1000"}`,
			},
		},
		{
			name:      "client1 relay preset kill",
			client:    client1,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:55.875584+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"DELETE","request_url":"http://localhost:9080/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.875946+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.875984+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayKillWidget.func1","file":"/tests/rest/handler.go","line":246},"msg":"handling RelayKillWidget request","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876007+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":126},"msg":"outbound call - request","service_name":"first-service","request_headers":{},"request_method":"DELETE","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.876192+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.876224+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.87624+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62810","user_agent":"first-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876409+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":126},"msg":"outbound call - response","service_name":"first-service","call_duration":354875,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:09:55.876425+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":131},"msg":"KillWidget request sent","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9081/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:55.876442+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-1","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876569+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":895709,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:09:55.876585+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayKillWidget","file":"/tests/rest/client.go","line":217},"msg":"RelayKillWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/relay/widget/widget_1000"}`,
			},
		},

		{
			name:      "client1 direct preset create",
			client:    client1,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:13:00.008925+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"POST","request_url":"http://localhost:9080/widget","request_body":""}`,
				`{"time":"2026-09-24T14:13:00.00915+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:13:00.009175+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:13:00.009187+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:13:00.009313+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":318542,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:13:00 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:13:00.009327+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget"}`,
			},
		},
		{
			name:      "client1 direct preset find",
			client:    client1,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:14:21.890271+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"GET","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.890403+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.890412+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.890421+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.890514+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":191667,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:14:21 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:14:21.890527+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
		},
		{
			name:      "client1 direct preset update",
			client:    client1,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:14:21.892002+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"PUT","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.892239+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.892254+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.892275+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.892425+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":354042,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:14:21 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:14:21.892453+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
		},
		{
			name:      "client1 direct preset kill",
			client:    client1,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:15:31.224784+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"DELETE","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:15:31.224856+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:15:31.22486+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:15:31.224865+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:15:31.224919+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":108042,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:15:31 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:15:31.224926+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":131},"msg":"KillWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
		},

		{
			name:      "client1 direct dynamic create",
			client:    client1,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:17:05.86093+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["5abc7029-825c-4ae6-8fe1-cc70a391c4eb"]},"request_method":"POST","request_url":"http://localhost:9080/widget","request_body":""}`,
				`{"time":"2026-09-24T14:17:05.861088+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb","request_body":""}`,
				`{"time":"2026-09-24T14:17:05.861097+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb"}`,
				`{"time":"2026-09-24T14:17:05.861108+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-1","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb"}`,
				`{"time":"2026-09-24T14:17:05.861239+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":245792,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:17:05 GMT"],"X-Go11y-Requestid":["5abc7029-825c-4ae6-8fe1-cc70a391c4eb"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:17:05.861252+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client1 direct dynamic find",
			client:    client1,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:17:41.715732+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["7af7483c-a611-49ee-9226-4c07697fdd4d"]},"request_method":"GET","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:17:41.715891+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d","request_body":""}`,
				`{"time":"2026-09-24T14:17:41.715908+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d"}`,
				`{"time":"2026-09-24T14:17:41.715917+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-1","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d"}`,
				`{"time":"2026-09-24T14:17:41.71601+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":230584,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:17:41 GMT"],"X-Go11y-Requestid":["7af7483c-a611-49ee-9226-4c07697fdd4d"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:17:41.716035+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client1 direct dynamic update",
			client:    client1,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:20:15.004897+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"]},"request_method":"PUT","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:20:15.004989+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa","request_body":""}`,
				`{"time":"2026-09-24T14:20:15.004993+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"}`,
				`{"time":"2026-09-24T14:20:15.004998+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-1","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"}`,
				`{"time":"2026-09-24T14:20:15.005067+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":145542,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:20:15 GMT"],"X-Go11y-Requestid":["9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:20:15.005073+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client1 direct dynamic kill",
			client:    client1,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:21:31.757138+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-1"],"X-Go11y-Requestid":["91aa0f09-85f2-4a62-8e8d-305235c2b68f"]},"request_method":"DELETE","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:21:31.757225+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f","request_body":""}`,
				`{"time":"2026-09-24T14:21:31.757228+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f"}`,
				`{"time":"2026-09-24T14:21:31.757232+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-1","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f"}`,
				`{"time":"2026-09-24T14:21:31.757305+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":143833,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:21:31 GMT"],"X-Go11y-Requestid":["91aa0f09-85f2-4a62-8e8d-305235c2b68f"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:21:31.757311+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":127},"msg":"KillWidget request sent","client_name":"go11y_test","url":"http://localhost:9080/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},

		{
			name:      "client2 relay preset create",
			client:    client2,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:06:29.753922+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"POST","request_url":"http://localhost:9081/relay/widget","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.764536+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-2","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.764584+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayCreateWidget.func1","file":"/tests/rest/handler.go","line":218},"msg":"handling RelayCreateWidget request","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-2","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.764656+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":104},"msg":"outbound call - request","service_name":"second-service","request_headers":{},"request_method":"POST","request_url":"http://localhost:9080/widget","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.766543+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62645","user_agent":"second-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:06:29.766613+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:62645","user_agent":"second-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.766656+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62645","user_agent":"second-server-client","method":"POST","path":"/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.766962+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":104},"msg":"outbound call - response","service_name":"second-service","call_duration":2182125,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:06:29 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:06:29.767001+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-2","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9080/widget"}`,
				`{"time":"2026-09-24T14:06:29.76706+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62644","user_agent":"test-client-2","method":"POST","path":"/relay/widget"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:06:29.767212+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":12775625,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:06:29 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:06:29.767241+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayCreateWidget","file":"/tests/rest/client.go","line":197},"msg":"RelayCreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/relay/widget"}`,
			},
		},

		{
			name:      "client2 relay preset find",
			client:    client2,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:21.140373+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"GET","request_url":"http://localhost:9081/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.14051+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-2","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140525+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayFindWidget.func1","file":"/tests/rest/handler.go","line":161},"msg":"handling RelayFindWidget request","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-2","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140538+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":57},"msg":"outbound call - request","service_name":"second-service","request_headers":{},"request_method":"GET","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140613+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62797","user_agent":"second-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:21.140629+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:62797","user_agent":"second-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140636+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62797","user_agent":"second-server-client","method":"GET","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140706+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":57},"msg":"outbound call - response","service_name":"second-service","call_duration":145292,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:21 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:09:21.140713+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-2","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9080/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:21.140754+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62796","user_agent":"test-client-2","method":"GET","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:21.140842+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":428959,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:21 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:09:21.14085+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayFindWidget","file":"/tests/rest/client.go","line":154},"msg":"RelayFindWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/relay/widget/widget_1000"}`,
			},
		},
		{
			name:      "client2 relay preset update",
			client:    client2,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:55.869954+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"PUT","request_url":"http://localhost:9081/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870258+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870272+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayUpdateWidget.func1","file":"/tests/rest/handler.go","line":190},"msg":"handling RelayUpdateWidget request","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870293+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":79},"msg":"outbound call - request","service_name":"second-service","request_headers":{},"request_method":"PUT","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870424+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.870434+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870451+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"PUT","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870583+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":79},"msg":"outbound call - response","service_name":"second-service","call_duration":244625,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:09:55.870603+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9080/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:55.870626+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"PUT","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.870772+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":727375,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:09:55.870796+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayUpdateWidget","file":"/tests/rest/client.go","line":174},"msg":"RelayUpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/relay/widget/widget_1000"}`,
			},
		},
		{
			name:      "client2 relay preset kill",
			client:    client2,
			route:     "relay",
			requestID: uuid.MustParse("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:09:55.875584+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"request_method":"DELETE","request_url":"http://localhost:9081/relay/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.875946+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.875984+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).RelayKillWidget.func1","file":"/tests/rest/handler.go","line":246},"msg":"handling RelayKillWidget request","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876007+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":126},"msg":"outbound call - request","service_name":"second-service","request_headers":{},"request_method":"DELETE","request_url":"http://localhost:9080/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.876192+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","request_body":""}`,
				`{"time":"2026-09-24T14:09:55.876224+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.87624+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62810","user_agent":"second-server-client","method":"DELETE","path":"/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876409+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":126},"msg":"outbound call - response","service_name":"second-service","call_duration":354875,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:09:55.876425+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":131},"msg":"KillWidget request sent","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","url":"http://localhost:9080/widget/widget_1000"}`,
				`{"time":"2026-09-24T14:09:55.876442+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62809","user_agent":"test-client-2","method":"DELETE","path":"/relay/widget/widget_1000"},"request_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
				`{"time":"2026-09-24T14:09:55.876569+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":895709,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:09:55 GMT"],"X-Go11y-Requestid":["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:09:55.876585+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).RelayKillWidget","file":"/tests/rest/client.go","line":217},"msg":"RelayKillWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/relay/widget/widget_1000"}`,
			},
		},

		{
			name:      "client2 direct preset create",
			client:    client2,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:13:00.008925+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"POST","request_url":"http://localhost:9081/widget","request_body":""}`,
				`{"time":"2026-09-24T14:13:00.00915+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:13:00.009175+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:13:00.009187+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:62950","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:13:00.009313+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":318542,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:13:00 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:13:00.009327+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget"}`,
			},
		},
		{
			name:      "client2 direct preset find",
			client:    client2,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:14:21.890271+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"GET","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.890403+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.890412+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.890421+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.890514+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":191667,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:14:21 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:14:21.890527+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
		},
		{
			name:      "client2 direct preset update",
			client:    client2,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:14:21.892002+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"PUT","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.892239+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:14:21.892254+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.892275+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63072","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:14:21.892425+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":354042,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:14:21 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:14:21.892453+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
		},
		{
			name:      "client2 direct preset kill",
			client:    client2,
			route:     "direct",
			requestID: uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:15:31.224784+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"request_method":"DELETE","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:15:31.224856+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa","request_body":""}`,
				`{"time":"2026-09-24T14:15:31.22486+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:15:31.224865+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63112","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"66666666-7777-8888-9999-aaaaaaaaaaaa"}`,
				`{"time":"2026-09-24T14:15:31.224919+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":108042,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:15:31 GMT"],"X-Go11y-Requestid":["66666666-7777-8888-9999-aaaaaaaaaaaa"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:15:31.224926+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":131},"msg":"KillWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
		},

		{
			name:      "client2 direct dynamic create",
			client:    client2,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "create",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:17:05.86093+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["5abc7029-825c-4ae6-8fe1-cc70a391c4eb"]},"request_method":"POST","request_url":"http://localhost:9081/widget","request_body":""}`,
				`{"time":"2026-09-24T14:17:05.861088+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb","request_body":""}`,
				`{"time":"2026-09-24T14:17:05.861097+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).CreateWidget.func1","file":"/tests/rest/handler.go","line":115},"msg":"handling CreateWidget request","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb"}`,
				`{"time":"2026-09-24T14:17:05.861108+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63204","user_agent":"test-client-2","method":"POST","path":"/widget"},"request_id":"5abc7029-825c-4ae6-8fe1-cc70a391c4eb"}`,
				`{"time":"2026-09-24T14:17:05.861239+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":245792,"status_code":200,"response_headers":{"Content-Length":["97"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:17:05 GMT"],"X-Go11y-Requestid":["5abc7029-825c-4ae6-8fe1-cc70a391c4eb"]},"response_body":"eyJtZXNzYWdlIjoiQ3JlYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTIzNDU2Nzg5QUJDIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTIzNDU2Nzg5QUJDIn19"}`,
				`{"time":"2026-09-24T14:17:05.861252+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).CreateWidget","file":"/tests/rest/client.go","line":109},"msg":"CreateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client2 direct dynamic find",
			client:    client2,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "find",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:17:41.715732+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["7af7483c-a611-49ee-9226-4c07697fdd4d"]},"request_method":"GET","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:17:41.715891+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d","request_body":""}`,
				`{"time":"2026-09-24T14:17:41.715908+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).FindWidget.func1","file":"/tests/rest/handler.go","line":70},"msg":"handling FindWidget request","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d"}`,
				`{"time":"2026-09-24T14:17:41.715917+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63220","user_agent":"test-client-2","method":"GET","path":"/widget/widget_1000"},"request_id":"7af7483c-a611-49ee-9226-4c07697fdd4d"}`,
				`{"time":"2026-09-24T14:17:41.71601+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":230584,"status_code":200,"response_headers":{"Content-Length":["79"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:17:41 GMT"],"X-Go11y-Requestid":["7af7483c-a611-49ee-9226-4c07697fdd4d"]},"response_body":"eyJtZXNzYWdlIjoiRm91bmQgd2lkZ2V0IHdpdGggSUQgd2lkZ2V0XzEwMDAiLCJ3aWRnZXRJRCI6eyJpZCI6IndpZGdldF8xMDAwIn19"}`,
				`{"time":"2026-09-24T14:17:41.716035+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).FindWidget","file":"/tests/rest/client.go","line":62},"msg":"FindWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client2 direct dynamic update",
			client:    client2,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "update",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:20:15.004897+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"]},"request_method":"PUT","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:20:15.004989+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa","request_body":""}`,
				`{"time":"2026-09-24T14:20:15.004993+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).UpdateWidget.func1","file":"/tests/rest/handler.go","line":93},"msg":"handling UpdateWidget request","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"}`,
				`{"time":"2026-09-24T14:20:15.004998+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63338","user_agent":"test-client-2","method":"PUT","path":"/widget/widget_1000"},"request_id":"9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"}`,
				`{"time":"2026-09-24T14:20:15.005067+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":145542,"status_code":200,"response_headers":{"Content-Length":["81"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:20:15 GMT"],"X-Go11y-Requestid":["9e4df4f6-9bc6-4c06-be08-a4f28e73c1aa"]},"response_body":"eyJtZXNzYWdlIjoiVXBkYXRlZCB3aWRnZXQgd2l0aCBJRCB3aWRnZXRfMTAwMCIsIndpZGdldElEIjp7ImlkIjoid2lkZ2V0XzEwMDAifX0="}`,
				`{"time":"2026-09-24T14:20:15.005073+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).UpdateWidget","file":"/tests/rest/client.go","line":84},"msg":"UpdateWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
		{
			name:      "client2 direct dynamic kill",
			client:    client2,
			route:     "direct",
			requestID: uuid.Nil(),
			request:   "kill",
			expectedLogs: []string{
				`{"time":"2026-09-24T14:21:31.757138+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - request","client_name":"go11y_test","request_headers":{"User-Agent":["test-client-2"],"X-Go11y-Requestid":["91aa0f09-85f2-4a62-8e8d-305235c2b68f"]},"request_method":"DELETE","request_url":"http://localhost:9081/widget/widget_1000","request_body":""}`,
				`{"time":"2026-09-24T14:21:31.757225+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":223},"msg":"request received","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f","request_body":""}`,
				`{"time":"2026-09-24T14:21:31.757228+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*restHandler).KillWidget.func1","file":"/tests/rest/handler.go","line":138},"msg":"handling KillWidget request","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f"}`,
				`{"time":"2026-09-24T14:21:31.757232+08:00","level":"DEBUG","source":{"function":"github.com/jsnfwlr/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":238},"msg":"request processed","origin":{"client_ip":"[::1]:63378","user_agent":"test-client-2","method":"DELETE","path":"/widget/widget_1000"},"request_id":"91aa0f09-85f2-4a62-8e8d-305235c2b68f"}`,
				`{"time":"2026-09-24T14:21:31.757305+08:00","level":"INFO","source":{"function":"net/http.(*Client).do","file":"src/net/http/client.go","line":745},"msg":"outbound call - response","client_name":"go11y_test","call_duration":143833,"status_code":200,"response_headers":{"Content-Length":["80"],"Content-Type":["application/json"],"Date":["Thu, 24 Sep 2026 06:21:31 GMT"],"X-Go11y-Requestid":["91aa0f09-85f2-4a62-8e8d-305235c2b68f"]},"response_body":"eyJtZXNzYWdlIjoiS2lsbGVkIHdpZGdldCB3aXRoIElEIHdpZGdldF8xMDAwIiwid2lkZ2V0SUQiOnsiaWQiOiJ3aWRnZXRfMTAwMCJ9fQ=="}`,
				`{"time":"2026-09-24T14:21:31.757311+08:00","level":"INFO","source":{"function":"github.com/jsnfwlr/go11y/tests/rest.(*Client).KillWidget","file":"/tests/rest/client.go","line":127},"msg":"KillWidget request sent","client_name":"go11y_test","url":"http://localhost:9081/widget/widget_1000"}`,
			},
			allowedErrorRegexp: []*regexp.Regexp{
				regexp.MustCompile(`ERR 0-root\) field request_headers\.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
				regexp.MustCompile(`ERR 4-root\) field response_headers.X-Go11y-Requestid: expected '\[[^\]]+\]', received '[^\']+'`),
			},
		},
	}

	time.Sleep(1 * time.Second)

	// output := bufOut.String()
	// t.Logf("Test ready. Output:\n%s", output)
	bufOut.Reset()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.requestID != uuid.Nil() {
				ctx = context.WithValue(ctx, go11y.RequestIDInstance, tc.requestID.String())
			} else {
				ctx = context.WithValue(ctx, go11y.RequestIDInstance, "")
			}

			switch tc.route {
			case "relay":
				switch tc.request {
				case "create":
					resp, err := tc.client.RelayCreateWidget(ctx)
					if err != nil {
						t.Errorf("RelayCreateWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "find":
					resp, err := tc.client.RelayFindWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("RelayFindWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "update":
					resp, err := tc.client.RelayUpdateWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("RelayUpdateWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "kill":
					resp, err := tc.client.RelayKillWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("RelayKillWidget failed: %v", err)
					}

					defer resp.Body.Close()
				}

			case "direct":
				switch tc.request {
				case "create":

					resp, err := tc.client.CreateWidget(ctx)
					if err != nil {
						t.Errorf("CreateWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "find":
					resp, err := tc.client.FindWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("FindWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "update":
					resp, err := tc.client.UpdateWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("UpdateWidget failed: %v", err)
					}

					defer resp.Body.Close()

				case "kill":
					resp, err := tc.client.KillWidget(ctx, "widget_1000")
					if err != nil {
						t.Errorf("KillWidget failed: %v", err)
					}

					defer resp.Body.Close()
				}
			}

			errs := compareLogs(t, bufOut, tc.expectedLogs, tc.name, fmt.Sprintf("go11y_test_%s.tmp", strings.ReplaceAll(tc.name, " ", "_")), tc.allowedErrorRegexp)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
