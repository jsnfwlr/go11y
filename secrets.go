package go11y

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

var (
	forbiddenKeysRex = regexp.MustCompile(`(?i)(authorization|authorisation|cookie|password|secret|key|token)`)
	falsePositives   = []string{"authorizationDate", "authorisationDate", "authorizationType", "authorisationType"}
)

// RedactSecret converts secrets to character-length-character notation, with variable length for the number of
// characters to reveal on each side, up to a maximum of an eighth on each side.
// Minimum secret length is to get character-length-character notation is 8, below that but above 4 characters in length
// or if the reveal is set to 0, the secret will be redacted with *-length-*
// examples:
// with a reveal value of 1 - "accessibility" becomes "a[11]y"
//
//	"internationalisation" becomes "i[18]n"
//	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij" becomes "A[34]j"
//	"ABCD" becomes "****"
//	"ABCDE" becomes "*3*"
//	"ABCDEF" becomes "*4*"
//	"ABCDEFG" becomes "*5*"
//	"ABCDEFGH" becomes "A[6]H"
//
// with a reveal value of 2 - "observability" remains "o[11]y"
//
//	"internationalisation" becomes "in[16]on"
//	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij" becomes "AB[32]ij"
//
// with a reveal value of 4 - "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij" becomes "ABCD[28]ghij"
// See ./config_test.go for more examples
func RedactSecret(secretStr string, reveal int) string {
	if reveal > (len(secretStr) / 8) {
		reveal = len(secretStr) / 8
	}

	switch {
	case len(secretStr) == 0:
		return ""
	case len(secretStr) < 5: // below 5 characters there isn't enough to redact without revealing too much, just show *
		return strings.Repeat("*", len(secretStr))
	case len(secretStr) <= 7 || reveal == 0:
		return fmt.Sprintf("*%d*", len(secretStr)-2)
	default:
		return fmt.Sprintf("%s[%d]%s", secretStr[0:reveal], len(secretStr)-(reveal*2), secretStr[(len(secretStr)-reveal):])
	}
}

// RedactHeaders redacts sensitive information from HTTP headers such as Authorization and Cookie.
// It applies RedactSecret to the values of these headers.
func RedactHeaders(headers http.Header) http.Header {
	redactedHeaders := make(http.Header)
	for key, values := range headers {
		if forbiddenKeysRex.MatchString(key) && !slices.Contains(falsePositives, key) {
			for i := range values {
				if len(redactedHeaders[key]) == 0 {
					redactedHeaders[key] = make([]string, len(values))
				}

				redactedHeaders[key][i] = RedactSecret(values[i], 6)
			}
		} else {
			redactedHeaders[key] = values
		}
	}

	return redactedHeaders
}

// RedactBody redacts sensitive information from a JSON body.
func RedactBody(jsonBlob []byte) []byte {
	body := map[string]any{}

	err := json.Unmarshal(jsonBlob, &body)
	if err != nil {
		return jsonBlob
	}

	body = redactFields(body)

	jsonBlob, err = json.Marshal(body)
	if err != nil {
		return jsonBlob
	}

	return jsonBlob
}

func redactFields(field map[string]any) map[string]any {
	for key, value := range field {
		if forbiddenKeysRex.MatchString(key) && !slices.Contains(falsePositives, key) {
			nv := RedactSecret(fmt.Sprintf("%v", value), 6)
			field[key] = nv
		}

		switch v := value.(type) {
		case map[string]any:
			field[key] = redactFields(v)
		}
	}
	return field
}
