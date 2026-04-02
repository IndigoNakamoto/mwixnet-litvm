# NOSTR_EVENTS (archived)

This filename used to hold an experimental **30000-range** kind table. **Normative** MLN Nostr wire (kinds **31250** and **31251**, `content` JSON, `nostrKeyHash` binding, NIP-33 `d` tags) is defined only in [`NOSTR_MLN.md`](NOSTR_MLN.md).

- **CI fixtures:** [`nostr/fixtures/`](../nostr/fixtures/) + [`nostr/validate_fixtures.py`](../nostr/validate_fixtures.py)
- **Scripts:** [`scripts/mln_nostr_wire.py`](../scripts/mln_nostr_wire.py), [`scripts/publish_grievance.py`](../scripts/publish_grievance.py), [`scripts/listen_makers.py`](../scripts/listen_makers.py), [`scripts/listen_grievances.py`](../scripts/listen_grievances.py), [`scripts/mln-nostr-demo.py`](../scripts/mln-nostr-demo.py)

The retired kind numbers (**30001** maker ad, **31001** grievance pointer) exist only in git history if you need them for archaeology.
