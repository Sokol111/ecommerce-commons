# AGENTS.md

## What this is

`ecommerce-commons` is the shared Go library imported by every service in the ecommerce
workspace (catalog, product-query, category-query, image, tenant). It has **no `cmd/` and no
`main`**. It ships reusable building blocks plus their Fx wiring, which services compose in
their own `cmd/main.go`. There is nothing to "run"; you build, test, and lint it, and consumers
pick up changes locally through the root `go.work` (no tag/release needed for local dev — see the
workspace root `AGENTS.md`).

## Commands

```bash
make test                # all tests: -v -race -coverprofile
make test-unit           # -short (unit only)
make test-integration    # -tags=integration (spins up real deps via testcontainers — needs Docker)
make lint                # golangci-lint (v2; stricter-than-services config in .golangci.yml)
make fmt                 # gofmt -s + goimports
make generate-mocks      # mockery (not currently used)
make vuln-check          # govulncheck (excludes testutil/mocks/test)
make check-all           # deps + fmt + lint + test + vuln-check (the CI pipeline)
make install-tools       # golangci-lint, mockery, govulncheck, go-mod-outdated, go-licenses
```

Run a single test: `go test ./pkg/kafka/consumer/ -run TestRouter -v`
Integration tests are gated behind `-tags=integration`; plain `go test ./...` / `make test-unit` skips them.

## Module system (the core convention)

Runtime packages contain framework-agnostic functionality. Their Fx constructors and options
live in a sibling `fxconfig` package: for example, `pkg/mongo` is wired by
`pkg/mongo/fxconfig`, and configuration loading by `pkg/core/config/fxconfig`. Do not add Fx
imports or DI constructors to runtime packages. Add them to the closest `fxconfig` package.

Each Fx package exposes a constructor returning `fx.Option`, conventionally named
`New*Module()`. `pkg/fxconfig.NewCommonsModule()` aggregates the standard library modules:
core, HTTP, Mongo, observability, Kafka, tenant, client credentials, and JWKS validation.
Services can compose this aggregate or individual `fxconfig` modules; they do not construct
infrastructure components by hand.

Modules load their configuration from koanf through the shared configuration loader. Tests that
compose Fx modules must provide the required configuration through the normal configuration path.

Top-level modules and what they wire:
- `pkg/core/fxconfig` — `NewCoreModule()`: config loading (koanf + dotenv), zap logger, readiness
  health, and 5-minute Fx start/stop timeouts. This is the foundation every service starts with.
- `pkg/kafka/fxconfig` — `NewKafkaModule()`: Kafka configuration, producer, protobuf serde, and
  the outbox pattern. Consumer wiring is opt-in via `pkg/kafka/consumer/fxconfig`.
- `pkg/mongo/fxconfig` — `NewMongoModule()`: Mongo client, database, transaction manager, metric
  views, lifecycle, and the default single-database migration runner.
- `pkg/tenant/fxconfig` — `NewTenantModule()`: multi-tenancy lifecycle, migrations, cleanup worker,
  and Connect-RPC interceptors.
- `pkg/http/fxconfig`, `pkg/observability/fxconfig`, and `pkg/security/*/fxconfig` — transport,
  telemetry, and authentication building blocks.

Config loading is centralized in `config.Load[T]` (`pkg/core/config/loader.go`): each module loads
its own subtree of the service's YAML by key (`"mongo"`, `"kafka"`, etc.).

## Multi-tenancy (database-per-tenant)

Tenant data lives in a **separate Mongo database per tenant**, named `{baseDatabase}_{slug}`.
The slug is resolved per-request from context. This is the single most important cross-cutting
concern to get right in repository code:

- `mongo.NewGenericRepository[Domain, Entity](...)` accepts a `mongo.CollectionProvider`. The
  provider resolves a collection for the request context, so services can supply either a fixed
  collection or tenant-aware collection selection without changing repository code.
- `tenant.NewMultiTenantCollectionProvider(...)` — tenant-aware collection selection. It resolves
  the database from the tenant in request context and is required for tenant-scoped data.
- `mongo.NewStaticCollectionProvider(...)` — fixed collection in the base database, suitable for
  non-tenant data such as the transactional `outbox`.

Choosing the wrong provider silently reads/writes the wrong database, so match it to whether the
data is tenant-scoped. See `pkg/mongo/collection_provider.go` and
`generic_repository.go`.

## Connect-RPC interceptor ordering

Interceptors are collected via the fx group `connect_interceptor` and sorted by an integer
`Priority` (**lower runs earlier**). Because ordering is global across packages, the priorities
are coordinated constants — when adding an interceptor, pick a priority that fits this chain:

```
10 Recovery  15 Tracing  18 Tenant-Resolver  20 Logger  22 Auth
25 Validation  26 Tenant-Validator  30 Timeout  40 RateLimit  50 Bulkhead
```

The named constants live in `pkg/tenant/fxconfig/module.go`
(`ResolverInterceptorPriority=18`, `ValidatorInterceptorPriority=26`) and
`pkg/security/validation/fxconfig/module.go` (`AuthInterceptorPriority=22`). Tracing uses 15 in
`pkg/observability/fxconfig/module.go`; the built-in chain is in
`pkg/http/connect/interceptor/fxconfig/module.go`. Tenant/logger/auth ordering is deliberate:
resolve tenant before logging so the tenant field appears in logs, validate auth after logging so
failures are logged.

## Linting notes

`.golangci.yml` is intentionally strict for a library: `godot` (comments end with a period),
`revive`'s `exported` rule (exported symbols need doc comments), `gocyclo` at complexity 20,
plus `gosec` and `errcheck` with `check-blank`/`check-type-assertions`. Test files relax several
of these. Keep exported symbols documented with period-terminated comments to pass CI.
