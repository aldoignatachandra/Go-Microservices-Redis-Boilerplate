package utils

import (
	"context"

	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// ApplyRequestMetadataToEvent copies request metadata from context to event metadata.
func ApplyRequestMetadataToEvent(ctx context.Context, event *eventbus.Event) {
	if ctx == nil || event == nil {
		return
	}

	correlationID := GetCorrelationIDFromContext(ctx)
	if correlationID == "" {
		legacyCorrelationID, _ := ctx.Value("correlation_id").(string)
		correlationID = legacyCorrelationID
	}
	if correlationID != "" {
		event.WithCorrelationID(correlationID)
	}

	if requestID := GetRequestIDFromContext(ctx); requestID != "" {
		event.WithMetadata("request_id", requestID)
	}
	if ipAddress := GetIPAddressFromContext(ctx); ipAddress != "" {
		event.WithMetadata("ip_address", ipAddress)
	}
	if userAgent := GetUserAgentFromContext(ctx); userAgent != "" {
		event.WithMetadata("user_agent", userAgent)
	}
}
