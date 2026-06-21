package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := otelhttp.NewHandler(next, serviceName)
		return handler
	}
}

func AddSpanAttributes(r *http.Request, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(r.Context())
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

func RecordSpanError(r *http.Request, err error) {
	span := trace.SpanFromContext(r.Context())
	if span != nil {
		span.RecordError(err)
	}
}
