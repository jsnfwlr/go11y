package go11y

import (
	"net/http"
)

// HTTPWriter is a wrapper around http.ResponseWriter that allows us to capture the response body for logging purposes.
// It implements the http.ResponseWriter interface and optionally the http.Flusher interface if the underlying writer
// supports it.
type HTTPWriter struct {
	http       http.ResponseWriter // wrap an existing writer
	statusCode int                 // capture the status code for logging
	body       []byte              // capture the response body for logging
}

// Header returns the header map that will be sent by WriteHeader.
func (w *HTTPWriter) Header() http.Header {
	return w.http.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *HTTPWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...) // capture the response body for logging
	return w.http.Write(data)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *HTTPWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode // capture the status code for logging
	w.http.WriteHeader(statusCode)
}

// HTTPWriterFlusher is a wrapper around HTTPWriter that also implements the http.Flusher interface if the underlying
// http.ResponseWriter supports it. This allows us to use the Flush method to flush the response buffer when needed.
type HTTPWriterFlusher struct {
	*HTTPWriter  // wrap our "normal" writer
	http.Flusher // keep a ref to the wrapped Flusher
}

// Flush sends any buffered data to the client.
func (w *HTTPWriterFlusher) Flush() {
	w.Flusher.Flush()
}

// NewHTTPWriter creates a new HTTPWriter that wraps the provided http.ResponseWriter. If the underlying writer
func NewHTTPWriter(w http.ResponseWriter) http.ResponseWriter {
	httpWriter := &HTTPWriter{
		http: w,
	}

	if flusher, ok := w.(http.Flusher); ok {
		return &HTTPWriterFlusher{
			HTTPWriter: httpWriter,
			Flusher:    flusher,
		}
	}

	return httpWriter
}
