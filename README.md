# mwixnet-litvm

**MLN stack (working title):** Litecoin [**MWEB**](https://github.com/litecoin-project/litecoin) for CoinSwap-style mixing, [**LitVM**](https://docs.litvm.com/) for registry / stake / slashing / grievances, [**Nostr**](https://nostr.com/) for discovery and gossip, **Tor** for transport.

### What we’re aiming for

We want a **trust-minimized** path for MWEB users to run [**CoinSwap-style**](https://forum.grin.mw/t/mimblewimble-coinswap-proposal/8322) mixing **without a single human coordinator** carrying authority. **Privacy-critical execution** stays in MWEB; **economic security and programmable rules** (staking, bonds, slashing, grievances) live on LitVM; **discovery and transport** use Nostr and Tor so high-frequency metadata does not land on expensive permanent L2 state or link peers at the IP layer. This repo is where that design is written down and iterated—**not** a shipped product yet.

This tree holds the **product specification**, research notes, and Cursor project configuration. **Status:** early-stage (spec v0.1, draft); no production implementation in-tree.

## Documentation

| Document | Purpose |
| -------- | ------- |
| [`PRODUCT_SPEC.md`](PRODUCT_SPEC.md) | Architecture, economics, phased roadmap, MWEB appendix (§14), LitVM grievance preimage (§13), open questions |
| [`AGENTS.md`](AGENTS.md) | Short agent / contributor orientation (layer boundaries, where truth lives) |
| [`research/COINSWAPD_TEARDOWN.md`](research/COINSWAPD_TEARDOWN.md) | Structural map of [ltcmweb/coinswapd](https://github.com/ltcmweb/coinswapd) (RPCs, onion shape, `ltcd` boundary) |

## Local reference code (optional)

To trace the **coinswapd** reference implementation, clone your fork or upstream next to the spec:

```bash
git clone https://github.com/ltcmweb/coinswapd.git research/coinswapd
```

The path `research/coinswapd/` is **gitignored**; it is not part of this repository.

## Cursor

Project rules live under [`.cursor/rules/`](.cursor/rules/) (e.g. MLN architecture). Skills under [`.cursor/skills/`](.cursor/skills/). See Cursor docs for Rules and Agent Skills.

## License

Not specified in this repo; add a `LICENSE` when you publish.
