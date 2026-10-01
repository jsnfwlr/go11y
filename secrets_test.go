package go11y

import (
	"fmt"
	"net/http"
	"testing"
)

func TestRedactSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		reveal int
		output string
	}{
		{
			input:  "",
			reveal: 1,
			output: "",
		},
		{
			input:  "a",
			reveal: 1,
			output: "*",
		},
		{
			input:  "ab",
			reveal: 1,
			output: "**",
		},
		{
			input:  "abc",
			reveal: 1,
			output: "***",
		},
		{
			input:  "abcd",
			reveal: 1,
			output: "****",
		},
		{
			input:  "abcde",
			reveal: 1,
			output: "*3*",
		},
		{
			input:  "abcdef",
			reveal: 1,
			output: "*4*",
		},
		{
			input:  "abcdefg",
			reveal: 1,
			output: "*5*",
		},
		{
			input:  "abcdefgh",
			reveal: 1,
			output: "a[6]h",
		},
		{
			input:  "abcdefghi",
			reveal: 1,
			output: "a[7]i",
		},
		{
			input:  "abcdefghij",
			reveal: 1,
			output: "a[8]j",
		},
		{
			input:  "abcdefghijk",
			reveal: 1,
			output: "a[9]k",
		},
		{
			input:  "abcdefghijkl",
			reveal: 1,
			output: "a[10]l",
		},
		{
			input:  "S2NE3iHSwP0XV47EEXL8mWmRdEfGscSJ+7EgePNgt0s=",
			reveal: 1,
			output: "S[42]=",
		},
		{
			input:  "golang.org/x/crypto v0.0.0-20180904163835-0709b304e793/go.mod h1:6SG95UA2DQfeDnfUPMdvaQW0Q7yPrPDi9nlGo2tz2b4=",
			reveal: 1,
			output: "g[107]=",
		},
		{
			input:  "internationalisation",
			reveal: 1,
			output: "i[18]n",
		},
		{
			input:  "kubes",
			reveal: 1,
			output: "*3*",
		},
		{
			input:  "kubernetes",
			reveal: 1,
			output: "k[8]s",
		},
		{
			input:  "accessibility",
			reveal: 1,
			output: "a[11]y",
		},
		{
			input:  "observability",
			reveal: 1,
			output: "o[11]y",
		},
		{
			input:  "Observability",
			reveal: 1,
			output: "O[11]y",
		},
		{
			input:  "translation",
			reveal: 1,
			output: "t[9]n",
		},
		{
			input:  "localisation",
			reveal: 1,
			output: "l[10]n",
		},
		{
			input:  "globalisation",
			reveal: 1,
			output: "g[11]n",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s to %s", tt.input, tt.output), func(t *testing.T) {
			got := RedactSecret(tt.input, tt.reveal)
			if tt.output != got {
				t.Errorf("Expected '%s' got '%s'", tt.output, got)
			}
		})
	}
}

func TestRedactHeaders(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input  http.Header
		output http.Header
	}{
		"no headers": {
			input:  http.Header{},
			output: http.Header{},
		},
		"non-sensitive headers": {
			input: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"Go-http-client/1.1"},
			},
			output: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"Go-http-client/1.1"},
			},
		},
		"short authorization header": {
			input: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer mysecrettoken"},
			},
			output: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Be[16]en"},
			},
		},
		"long authorization header": {
			input: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer myReallyReallyLooooooooooooooooooooooooooooooooongSecretToken"},
			},
			output: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer[56]tToken"},
			},
		},
		"broken cookie header": {
			input: http.Header{
				"Content-Type": []string{"application/json"},
				"Cookie": []string{
					"CF_Session=nk6bQNeJYdXw6N54O;",
					"Path=/;",
					"Secure;",
					"Expires=Thu, 27 Nov 2025 02:56:57 GMT;",
					"HttpOnly;",
					"SameSite=none",
				},
			},
			output: http.Header{
				"Content-Type": []string{"application/json"},
				"Cookie": []string{
					"CF_[23]4O;",
					"*5*",
					"*5*",
					"Expi[30]GMT;",
					"H[7];",
					"S[11]e",
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			redacted := RedactHeaders(tc.input)
			for key, expectedValues := range tc.output {
				actualValues, exists := redacted[key]
				if !exists {
					t.Errorf("Expected header %s to exist", key)
					continue
				}
				if len(actualValues) != len(expectedValues) {
					t.Errorf("Expected %d values for header %s, got %d", len(expectedValues), key, len(actualValues))
					continue
				}
				for i, expectedValue := range expectedValues {
					if actualValues[i] != expectedValue {
						t.Errorf("Expected header %s value %s, got %s", key, expectedValue, actualValues[i])
					}
				}
			}
		})
	}
}

func TestRedactBody(t *testing.T) {
	t.Parallel()

	jsonBlob := `{"email": "test-moose@postman.test.swooop.com.au","password": "DummyPasswordForTesting#2025","first_name": "Fester","last_name": "McTester","more": {"authorization": "redact me"},"email_verified": true,"external_id": "MM00001","authorizationDate": "2024-01-01","authorizationType": "KeyCard","monkey":"banana"}`
	expected := `{"authorizationDate":"2024-01-01","authorizationType":"KeyCard","email":"test-moose@postman.test.swooop.com.au","email_verified":true,"external_id":"MM00001","first_name":"Fester","last_name":"McTester","monkey":"*4*","more":{"authorization":"r[7]e"},"password":"Dum[22]025"}`

	received := RedactBody([]byte(jsonBlob))

	if string(received) != expected {
		t.Errorf("expected:\n\t%v\nreceived:\n\t%v", expected, string(received))
	}
}
