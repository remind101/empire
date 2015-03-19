package httpx

import "golang.org/x/net/context"

// WithRequestID inserts a RequestID into the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestID extracts a RequestID from a context.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}
