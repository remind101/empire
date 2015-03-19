package httpx

import "golang.org/x/net/context"

// RequestID represents a unique identifier for a request.
type RequestID string

// WithRequestID inserts a RequestID into the context.
func WithRequestID(ctx context.Context, requestID RequestID) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts a RequestID from a context.
func RequestIDFromContext(ctx context.Context) RequestID {
	requestID, _ := ctx.Value(requestIDKey).(RequestID)
	return requestID
}
