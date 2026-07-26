package fxconfig

import (
	"go.uber.org/fx"

	fx_config "github.com/Sokol111/ecommerce-commons/pkg/core/fxconfig"
	fx_http "github.com/Sokol111/ecommerce-commons/pkg/http/fxconfig"
	fx_kafka "github.com/Sokol111/ecommerce-commons/pkg/kafka/fxconfig"
	fx_mongo "github.com/Sokol111/ecommerce-commons/pkg/mongo/fxconfig"
	fx_observability "github.com/Sokol111/ecommerce-commons/pkg/observability/fxconfig"
	fx_token "github.com/Sokol111/ecommerce-commons/pkg/security/token/fxconfig"
	fx_validation "github.com/Sokol111/ecommerce-commons/pkg/security/validation/fxconfig"
	fx_tenant "github.com/Sokol111/ecommerce-commons/pkg/tenant/fxconfig"
)

func NewCommonsModule() fx.Option {
	return fx.Module("ecommerce-commons",
		fx_config.NewCoreModule(),
		fx_http.NewHTTPModule(),
		fx_mongo.NewMongoModule(),
		fx_observability.NewObservabilityModule(),
		fx_kafka.NewKafkaModule(),
		fx_tenant.NewTenantModule(),
		fx_token.NewClientCredentialsModule(),
		fx_validation.NewJWKSModule(),
	)
}
