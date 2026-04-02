# Optional: Docker Foundry (no local forge install)
FOUNDRY_IMAGE ?= ghcr.io/foundry-rs/foundry:latest
# This Makefile’s directory = repo root (do not use $(PWD)/contracts: wrong when invoking make from contracts/).
MK_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
CONTRACTS := $(MK_ROOT)/contracts

.PHONY: contracts-build contracts-test contracts-test-match contracts-fmt deploy-local

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
