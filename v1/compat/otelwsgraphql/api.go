// Package otelwsgraphql provides opentelemetry instrumentation for wsgraphql
package otelwsgraphql

import (
	"context"
	"regexp"

	"github.com/eientei/wsgraphql/v1"
	"github.com/eientei/wsgraphql/v1/apollows"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/eientei/wsgraphql/v1/compat/otelwsgraphql"
	instrumentationVersion = "1.0.0"
)

const (
	operationQuery        = "query"
	operationMutation     = "mutation"
	operationSubscription = "subscription"
)

// OperationOption provides customizations for operation interceptor
type OperationOption interface {
	applyOperation(*operationConfig)
}

// NewOperationInterceptor returns new otel-span reporting wsgraphql operation interceptor
func NewOperationInterceptor(options ...OperationOption) wsgraphql.InterceptorOperation {
	var c operationConfig

	defaultOptions := []OperationOption{
		WithSpanNameResolver(DefaultSpanNameResolver),
		WithSpanAttributesResolver(DefaultSpanAttributesResolver),
		WithStartSpanOptions(trace.WithSpanKind(trace.SpanKindServer)),
	}

	for _, o := range append(defaultOptions, options...) {
		o.applyOperation(&c)
	}

	return func(ctx context.Context, payload *apollows.PayloadOperation, handler wsgraphql.HandlerOperation) error {
		tracer := c.tracer

		if tracer == nil {
			if c.tracerProvider != nil {
				tracer = newTracer(c.tracerProvider)
			} else if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				tracer = newTracer(span.TracerProvider())
			} else {
				tracer = newTracer(otel.GetTracerProvider())
			}
		}

		opts := append(
			[]trace.SpanStartOption{trace.WithAttributes(c.attributesResolver(ctx, payload)...)},
			c.startSpanOptions...,
		)

		ctx, span := tracer.Start(ctx, c.nameResolver(ctx, payload), opts...)

		err := handler(ctx, payload)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		span.End()

		return err
	}
}

// SpanNameResolver determined span name from payload operation
type SpanNameResolver func(ctx context.Context, payload *apollows.PayloadOperation) string

// SpanAttributesResolver determines span attributes from payload operation
type SpanAttributesResolver func(ctx context.Context, payload *apollows.PayloadOperation) []attribute.KeyValue

// SpanASTAttributesResolver determines span attributes from payload operation after AST parsing
type SpanASTAttributesResolver func(ctx context.Context, payload *apollows.PayloadOperation) []attribute.KeyValue

// WithTracer provides predefined tracer instance
func WithTracer(tracer trace.Tracer) OperationOption {
	return optionFunc(func(c *operationConfig) {
		c.tracer = tracer
	})
}

// WithTracerProvider sets predefined tracer provider instance
func WithTracerProvider(tracerProvider trace.TracerProvider) OperationOption {
	return optionFunc(func(c *operationConfig) {
		c.tracerProvider = tracerProvider
	})
}

// WithStartSpanOptions provides extra starting span options
func WithStartSpanOptions(spanOptions ...trace.SpanStartOption) OperationOption {
	return optionFunc(func(c *operationConfig) {
		c.startSpanOptions = spanOptions
	})
}

// WithSpanNameResolver provides custom name resolver
func WithSpanNameResolver(resolver SpanNameResolver) OperationOption {
	return optionFunc(func(c *operationConfig) {
		c.nameResolver = resolver
	})
}

// WithSpanAttributesResolver provides custom attribute resolver
func WithSpanAttributesResolver(resolver SpanAttributesResolver) OperationOption {
	return optionFunc(func(c *operationConfig) {
		c.attributesResolver = resolver
	})
}

var queryRegex = regexp.MustCompile(`(query|mutation|subscription)\s*(\w*)`)

// DefaultSpanNameResolver default span name resolver function
func DefaultSpanNameResolver(_ context.Context, payload *apollows.PayloadOperation) string {
	parts := queryRegex.FindStringSubmatch(payload.Query)
	name := payload.OperationName
	kind := operationQuery

	if len(parts) == 3 {
		kind = parts[1]

		if name == "" {
			name = parts[2]
		}
	}

	if name == "" {
		return "gql." + kind
	}

	return "gql." + kind + "." + name
}

// DefaultSpanAttributesResolver default span attributes resolver function
func DefaultSpanAttributesResolver(
	_ context.Context,
	payload *apollows.PayloadOperation,
) (attrs []attribute.KeyValue) {
	parts := queryRegex.FindStringSubmatch(payload.Query)
	operationName := payload.OperationName
	kind := operationQuery

	if len(parts) == 3 {
		switch parts[1] {
		case operationSubscription, operationMutation:
			kind = parts[1]
		}

		if operationName == "" {
			operationName = parts[2]
		}
	}

	switch kind {
	case operationSubscription:
		attrs = append(attrs, semconv.GraphqlOperationTypeSubscription)
	case operationMutation:
		attrs = append(attrs, semconv.GraphqlOperationTypeMutation)
	case operationQuery:
		attrs = append(attrs, semconv.GraphqlOperationTypeQuery)
	}

	if operationName != "" {
		attrs = append(attrs, semconv.GraphqlOperationName(operationName))
	}

	attrs = append(attrs, semconv.GraphqlDocument(payload.Query))

	return
}

type operationConfig struct {
	nameResolver       SpanNameResolver
	attributesResolver SpanAttributesResolver
	tracer             trace.Tracer
	tracerProvider     trace.TracerProvider
	startSpanOptions   []trace.SpanStartOption
}

type optionFunc func(c *operationConfig)

func (o optionFunc) applyOperation(c *operationConfig) {
	o(c)
}

func newTracer(provider trace.TracerProvider) trace.Tracer {
	return provider.Tracer(instrumentationName, trace.WithInstrumentationVersion(instrumentationVersion))
}
