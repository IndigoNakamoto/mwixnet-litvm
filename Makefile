# Optional: Docker Foundry (no local forge install)
FOUNDRY_IMAGE ?= ghcr.io/foundry-rs/foundry:latest
CONTRACTS = $(PWD)/contracts

.PHONY: contracts-build contracts-test contracts-fmt deploy-local

contracts-build:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) build

contracts-test:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) test -vv

contracts-fmt:
	docker run --rm --entrypoint forge -v "$(CONTRACTS):/work" -w /work $(FOUNDRY_IMAGE) fmt

deploy-local:
	./scripts/deploy-local-anvil.sh
