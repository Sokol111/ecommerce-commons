# Event Generation Makefile
# Include this in your API project's Makefile:
#   include makefiles/eventgen.mk
#
# Or copy the targets directly into your Makefile.

# Configuration (override these in your Makefile)
EVENTGEN_PAYLOADS_DIR ?= avro
EVENTGEN_OUTPUT_DIR ?= gen/events
EVENTGEN_PACKAGE ?= events
EVENTGEN_ASYNCAPI ?= asyncapi/asyncapi.yaml
EVENTGEN_VERSION ?= latest

# Tools
AVROGEN ?= $(shell go env GOPATH)/bin/avrogen
EVENTGEN ?= $(shell go env GOPATH)/bin/eventgen

.PHONY: install-eventgen
install-eventgen: ## Install eventgen and avrogen tools
	@echo "Installing avrogen..."
	@go install github.com/hamba/avro/v2/cmd/avrogen@latest
	@echo "Installing eventgen..."
	@go install github.com/Sokol111/ecommerce-commons/cmd/eventgen@$(EVENTGEN_VERSION)

.PHONY: generate-events
generate-events: ## Generate event code from Avro payload schemas
	@echo "Generating event code..."
	@$(EVENTGEN) generate \
		--payloads $(EVENTGEN_PAYLOADS_DIR) \
		--output $(EVENTGEN_OUTPUT_DIR) \
		--package $(EVENTGEN_PACKAGE) \
		$(if $(wildcard $(EVENTGEN_ASYNCAPI)),--asyncapi $(EVENTGEN_ASYNCAPI)) \
		--verbose

.PHONY: validate-events
validate-events: ## Validate Avro payload schemas
	@$(EVENTGEN) validate --payloads $(EVENTGEN_PAYLOADS_DIR) --verbose

.PHONY: check-events
check-events: generate-events ## Generate and check for uncommitted changes
	@if ! git diff --quiet --exit-code $(EVENTGEN_OUTPUT_DIR); then \
		echo "Error: Generated event code is out of date. Run 'make generate-events' and commit the changes."; \
		git diff $(EVENTGEN_OUTPUT_DIR); \
		exit 1; \
	fi
	@echo "✓ Generated event code is up to date"
