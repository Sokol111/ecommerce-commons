# AGENTS.md

## What this is

`ecommerce-commons` is the shared Go library imported by every service in the ecommerce
workspace (catalog, product-query, category-query, image, tenant). It has **no `cmd/` and no
`main`** — it ships reusable building blocks, each exposed as an `fx` module that services
compose in their own `cmd/main.go`. There is nothing to "run"; you build, test, and lint it,
and consumers pick up changes locally through the root `go.work` (no tag/release needed for
local dev — see the workspace root `AGENTS.md`).

## Commands

```bash
make test                # all tests: -v -race -coverprofile
make test-unit           # -short (unit only)
make test-integration    # -tags=integration (spins up real deps via testcontainers — needs Docker)
make lint                # golangci-lint (v2; stricter-than-services config in .golangci.yml)
make fmt                 # gofmt -s + goimports
make generate-mocks      # mockery (not currently used — see "Mocks / test doubles" below)
make vuln-check          # govulncheck (excludes testutil/mocks/test)
make check-all           # deps + fmt + lint + test + vuln-check (the CI pipeline)
make install-tools       # golangci-lint, mockery, govulncheck, go-mod-outdated, go-licenses
```

Run a single test: `go test ./pkg/messaging/kafka/consumer/ -run TestRouter -v`
Integration tests are gated behind `-tags=integration`; plain `go test ./...` / `make test-unit` skips them.

## Module system (the core convention)

Every package exposes a constructor returning `fx.Option` — named `Module()`, `New*Module()`,
or `NewModule()`. Services assemble their app **only** from these; they never construct
components by hand. When adding a component, `fx.Provide` it inside the relevant module rather
than exporting a bare constructor.

Modules follow a **functional-options** pattern with a consistent testing escape hatch: the
production path loads config from koanf, while `With*Config(...)` options inject static config
for tests (e.g. `core.WithAppConfig`, `messaging.WithKafkaConfig`, `persistence.WithMongoConfig`,
plus `core.WithoutEnvFile()` / `core.WithoutConfigFile()`). Follow this pattern for any new module.

Top-level modules and what they wire:
- `pkg/core` — `NewCoreModule()`: config loading (koanf + dotenv), zap logger, readiness health.
  Also sets 5-min fx start/stop timeouts. This is the foundation every service starts with.
- `pkg/messaging` — `NewMessagingModule()`: Kafka producer, protobuf serde, and the outbox pattern.
- `pkg/persistence` — `NewPersistenceModule()`: MongoDB. `WithMigrations()` runs migrations on
  startup (single-tenant); omit it when the tenant module manages per-tenant migrations instead.
- `pkg/tenant` — `NewModule()`: multi-tenancy lifecycle + Connect-RPC interceptors.
- `pkg/http`, `pkg/grpc`, `pkg/security`, `pkg/observability`, `pkg/swaggerui` — transport, auth,
  and telemetry building blocks.

Config loading is centralized in `config.Load[T](k, "key", staticOverride)` (`pkg/core/config/
loader.go`): each module loads its own subtree of the service's YAML by key (`"mongo"`, `"kafka"`,
etc.), with the static-config option taking precedence when set.

## Multi-tenancy (database-per-tenant)

Tenant data lives in a **separate Mongo database per tenant**, named `{baseDatabase}_{slug}`.
The slug is resolved per-request from context. This is the single most important cross-cutting
concern to get right in repository code:

- `mongo.NewTenantRepository[Domain, Entity](...)` — tenant-scoped collections. Uses a
  `dynamicCollectionProvider` that picks the DB from a `DatabaseResolver` (the tenant module
  wires this to `MustSlugFromContext`, so a missing tenant in context is a hard failure).
- `mongo.NewGenericRepository[Domain, Entity](...)` — **non**-tenant collections in the base DB
  (e.g. the transactional `outbox`). Uses a `staticCollectionProvider`.

Choosing the wrong one silently reads/writes the wrong database, so match the constructor to
whether the data is tenant-scoped. See `pkg/persistence/mongo/collection_provider.go` and
`generic_repository.go`.

## Connect-RPC interceptor ordering

Interceptors are collected via the fx group `connect_interceptor` and sorted by an integer
`Priority` (**lower runs earlier**). Because ordering is global across packages, the priorities
are coordinated constants — when adding an interceptor, pick a priority that fits this chain:

```
10 Recovery  15 Tracing  18 Tenant-Resolver  20 Logger  22 Auth
25 Validation  26 Tenant-Validator  30 Timeout  40 RateLimit  50 Bulkhead
```

The named constants live in `pkg/tenant/module.go` (`ResolverInterceptorPriority=18`,
`ValidatorInterceptorPriority=26`), `pkg/security/validation/interceptor.go`
(`AuthInterceptorPriority=22`), and `pkg/observability/tracing/interceptor.go` (`=15`); the
built-in chain in `pkg/http/connect/interceptor/module.go`. Tenant/logger/auth ordering is
deliberate: resolve tenant before logging so the tenant field appears in logs, validate auth
after logging so failures are logged.

## Mocks / test doubles

Test doubles are **hand-written**, not generated: plain structs (often with a `sync.Mutex` and
per-method stub fields / func hooks), living in the same package's `_test.go` files. See
`pkg/messaging/patterns/outbox/repository_mock_test.go` for the established style, and follow it
for new tests. (The `make generate-mocks` target exists for possible future mockery use, but
there is no mockery config or generated mock in the tree today.)

## Linting notes

`.golangci.yml` is intentionally strict for a library: `godot` (comments end with a period),
`revive`'s `exported` rule (exported symbols need doc comments), `gocyclo` at complexity 20,
plus `gosec` and `errcheck` with `check-blank`/`check-type-assertions`. Test files relax several
of these. Keep exported symbols documented with period-terminated comments to pass CI.
