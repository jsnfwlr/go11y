package go11y

// This file contains utilities for managing requestIDs in HTTP requests and contexts.
// These are used by the go11y middleware and HTTP Transport wrappers to ensure consistent requestID handling between
// various microservices.

import (
	"context"
	"fmt"
	"net/http"
	"uuid"
)

type requestIDKey string

// RequestIDInstance is a constant for the context key used to store the requestID
const RequestIDInstance requestIDKey = "requestID"

// RequestIDHeader is a constant for the HTTP header used to store the requestID
const RequestIDHeader string = "X-Go11y-Requestid"

// GetContextRequestID retrieves the requestID from the context if possible otherwise it returns uuid.Nil() and an error
func GetContextRequestID(ctx context.Context) (requestID uuid.UUID, fault error) {
	if ctx == nil {
		return uuid.Nil(), fmt.Errorf("context is nil")
	}

	reqID := ctx.Value(RequestIDInstance)
	if reqID == nil {
		return uuid.Nil(), fmt.Errorf("requestID is missing from context")
	}

	reqIDStr, ok := reqID.(string)
	if !ok || reqIDStr == "" {
		return uuid.Nil(), fmt.Errorf("requestID in context is not a valid string")
	}

	requestID, err := uuid.Parse(reqIDStr)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("requestID in context is not a valid UUID")
	}

	return requestID, nil
}

// GetHTTPRequestID retrieves the requestID from the HTTP request headers if possible.
// This function exists for middleware to extract the requestID from incoming HTTP requests.
// If the request is nil or the requestID header is missing or invalid, an error is returned.
// Does not generate a new requestID if one is missing or invalid - that requires cloning the request and is beyond the
// scope of this function.
func GetHTTPRequestID(req *http.Request) (requestID uuid.UUID, fault error) {
	if req == nil {
		return uuid.Nil(), fmt.Errorf("request is nil")
	}
	reqID := req.Header.Get(RequestIDHeader)
	if reqID == "" {
		return uuid.Nil(), fmt.Errorf("requestID header is missing")
	}

	requestID, err := uuid.Parse(reqID)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("invalid requestID: %v", err)
	}

	return requestID, nil
}

// SetRequestIDFromContext checks if the provided HTTP request and context are valid, and that the request does not
// already have a requestID header set. If they are valid and the header is unset, it retrieves the requestID from the
// context and adds it to the headers of the request.
// NOTE: This is intended for use in hand-crafted client packages, use the AddRequestID Transporter from the go11y
// HTTPClient for openapi-codegen clients.
func SetRequestIDFromContext(ctx context.Context, req *http.Request) (ctxWithRequestID context.Context, fault error) {
	if req == nil {
		return ctx, fmt.Errorf("request is nil")
	}

	if ctx == nil {
		return ctx, fmt.Errorf("context is nil")
	}

	if req.Header.Get(RequestIDHeader) != "" {
		return ctx, nil
	}

	requestID, err := GetContextRequestID(ctx)
	if err != nil {
		requestID = uuid.New()
		ctx = context.WithValue(ctx, RequestIDInstance, requestID.String())
	}

	req.Header.Set(RequestIDHeader, requestID.String())
	return ctx, nil
}

// SetRequestIDFromRequest checks the HTTP request for a requestID header and, if present, adds it to the context.
// If the request is nil or the requestID header is missing or invalid, an error is returned.
func SetRequestIDFromRequest(ctx context.Context, req *http.Request) (ctxWithRequestID context.Context, fault error) {
	requestID, err := GetHTTPRequestID(req)
	if err != nil {
		return ctx, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctxWithRequestID = context.WithValue(ctx, RequestIDInstance, requestID.String())
	return ctxWithRequestID, nil
}
