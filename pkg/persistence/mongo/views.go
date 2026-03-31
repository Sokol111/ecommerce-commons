package mongo

import (
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// MetricViews returns SDK views that reduce cardinality of otelmongo metrics.
//
// Without these views, otelmongo generates high-cardinality series due to labels like
// network.peer.address, network.peer.port, db.namespace that provide little
// value but multiply series count significantly.
func MetricViews() []sdkmetric.View {
	return []sdkmetric.View{
		// Keep only db.operation.name and db.collection.name, drop network.peer.* and db.namespace.
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.client.operation.duration"},
			sdkmetric.Stream{
				AttributeFilter: attribute.NewAllowKeysFilter(
					semconv.DBOperationNameKey,
					semconv.DBCollectionNameKey,
				),
			},
		),
	}
}
