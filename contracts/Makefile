# Thin wrapper so `make` works from `contracts/` as well as repo root.
ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))..)

.PHONY: test test-golden contracts-build contracts-test contracts-test-match contracts-fmt deploy-local

# Shortcuts (Docker Foundry — no local `forge` required)
test: contracts-test
test-golden:
	@$(MAKE) contracts-test-match MATCH=EvidenceGoldenVectorsTest

contracts-test contracts-build contracts-fmt deploy-local:
	@$(MAKE) -C $(ROOT) $@

# Pass MATCH=..., e.g. `make contracts-test-match MATCH=EvidenceGoldenVectorsTest`
contracts-test-match:
	@$(MAKE) -C $(ROOT) contracts-test-match MATCH="$(MATCH)"
