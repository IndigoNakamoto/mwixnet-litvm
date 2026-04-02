# Optional: Docker Foundry (no local forge install)
FOUNDRY_IMAGE ?= ghcr.io/foundry-rs/foundry:latest
# This Makefile’s directory = repo root (do not use $(PWD)/contracts: wrong when invoking make from contracts/).
MK_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
CONTRACTS := $(MK_ROOT)/contracts

.PHONY: contracts-build contracts-test contracts-test-match contracts-fmt deploy-local test-grievance test-full-stack listen-makers listen-demo test-full-stack-with-nostr

# Optional: narrow tests, e.g. `make contracts-test-match MATCH=EvidenceGoldenVectorsTest`
MATCH ?=

contracts-build:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) build

contracts-test:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) test -vv

contracts-test-match:
	@test -n "$(MATCH)" || (echo "Set MATCH=ContractName, e.g. EvidenceGoldenVectorsTest" && exit 1)
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) test -vv --match-contract $(MATCH)

contracts-fmt:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) fmt

deploy-local:
	./scripts/deploy-local-anvil.sh

# Requires Anvil on ANVIL_RPC_URL + Docker; deploys then opens grievance with golden vectors.
test-grievance:
	./scripts/test-grievance-local.sh

test-full-stack:
	@echo "=== Running full grievance + Nostr stack test ==="
	./scripts/test-grievance-local.sh
	@echo "Grievance test passed. Publishing sample Nostr event (use your own privkey):"
	@echo "python3 scripts/publish_grievance.py 5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3 42 2d4d7ae96f39e2d5037f21782bc831874261ffe22743f74bbf865a39ec4df112 <YOUR_NOSTR_PRIVKEY_HEX>"
	@echo "Done. Phase 2 Nostr integration is now testable."

listen-makers:
	@echo "=== Building maker-ad subscription filters (kind 30001) ==="
	python3 scripts/listen_makers.py --help
	@echo "Tip: run with --stake 0x<registryAddress> to focus by litvm-stake tag."

listen-demo:
	python3 scripts/mln-nostr-demo.py --relay wss://relay.damus.io

test-full-stack-with-nostr:
	@echo "=== Full MLN Stack with live Nostr demo ==="
	@set -e; \
	ANVIL_CONTAINER=mln-anvil; \
	echo "Starting Docker Anvil container: $$ANVIL_CONTAINER"; \
	docker rm -f $$ANVIL_CONTAINER >/dev/null 2>&1 || true; \
	docker run --rm -d --name $$ANVIL_CONTAINER -p 8545:8545 --entrypoint anvil $(FOUNDRY_IMAGE) --host 0.0.0.0 --port 8545 >/dev/null; \
	trap 'echo "Stopping Docker Anvil container: $$ANVIL_CONTAINER"; docker stop $$ANVIL_CONTAINER >/dev/null 2>&1 || true' EXIT; \
	sleep 2; \
	./scripts/test-grievance-local.sh; \
	echo "Grievance test passed. Starting combined Nostr demo..."; \
	python3 scripts/mln-nostr-demo.py --relay wss://relay.damus.io --stake 0x000000000000000000000000000000000000bEef
