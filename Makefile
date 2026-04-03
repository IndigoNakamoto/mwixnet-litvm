# Optional: Docker Foundry (no local forge install)
FOUNDRY_IMAGE ?= ghcr.io/foundry-rs/foundry:latest
# This Makefile’s directory = repo root (do not use $(PWD)/contracts: wrong when invoking make from contracts/).
MK_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
CONTRACTS := $(MK_ROOT)/contracts

.PHONY: contracts-build contracts-test contracts-test-match contracts-fmt deploy-local test-grievance test-full-stack test-operator-smoke testnet-smoke build docker-build listen-makers listen-demo test-full-stack-with-nostr

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

# Golden NDJSON fixture + mlnd bridge + openGrievance on Anvil (no coinswapd). Requires Go 1.22+ on PATH.
test-operator-smoke:
	./scripts/mlnd-bridge-litvm-smoke.sh

# LitVM testnet (or any chain): set MLND_HTTP_URL + MLND_COURT_ADDR; see mlnd/.env.example and research/LITVM.md
testnet-smoke:
	./scripts/mlnd-testnet-smoke.sh

# CGO required (SQLite). Output: bin/mlnd
build:
	@mkdir -p "$(MK_ROOT)/bin"
	cd "$(MK_ROOT)/mlnd" && CGO_ENABLED=1 go build -o "$(MK_ROOT)/bin/mlnd" ./cmd/mlnd

docker-build:
	docker build -f "$(MK_ROOT)/mlnd/Dockerfile" -t mlnd:local "$(MK_ROOT)/mlnd"

test-full-stack:
	@echo "=== Running full grievance + Nostr stack test ==="
	./scripts/test-grievance-local.sh
	@echo "Grievance test passed. Sample kind-31251 pointer (use your privkey; evidence stays off Nostr):"
	@echo "python3 scripts/publish_grievance.py 0x5020b346b84d8c1da9aee82130e634fcbc120062e87eaaf9fe9f160bb921dcb3 42 <YOUR_NOSTR_PRIVKEY_HEX> \\"
	@echo "  --broadcast-json contracts/broadcast/Deploy.s.sol/31337/run-latest.json \\"
	@echo "  --accused 0x000000000000000000000000000000000000CAfE"
	@echo "Done. Phase 2 Nostr integration is now testable."

listen-makers:
	@echo "=== Maker-ad subscription filters (kind 31250, research/NOSTR_MLN.md) ==="
	python3 scripts/listen_makers.py --help
	@echo "Tip: pass --chain-id 31337 --maker 0x<litvm_maker> for an exact d-tag filter."

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
	python3 scripts/mln-nostr-demo.py --relay wss://relay.damus.io
