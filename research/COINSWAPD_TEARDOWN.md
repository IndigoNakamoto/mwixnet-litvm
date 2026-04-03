# `coinswapd` structural teardown (ltcmweb/coinswapd)

Source: in-repo fork at [`research/coinswapd/`](coinswapd/) (upstream: [ltcmweb/coinswapd](https://github.com/ltcmweb/coinswapd)). Hector Chu’s implementation tracks Tromp’s CoinSwap proposal; there is no README — this note maps **actual** entry points and types.

**Dependency reality:** This binary does **not** import [`mwebd`](https://github.com/ltcmweb/mwebd). It depends on **[`ltcmweb/ltcd`](https://github.com/ltcmweb/ltcd)** for MWEB wire types, `mw` primitives, and helpers under `ltcutil/mweb` (`CreateOutput`, `CreateKernel`, `SignOutput`, fee weights). For “where do Pedersen / bulletproofs live?” trace **`ltcd`** (`wire.MwebOutput`, `RangeProof.Verify`, etc.), not a separate `mwebd` import in this repo.

---

## 1. API and entry points

| Item | Location | Behavior |
|------|----------|----------|
| HTTP server | `main.go` | `go-ethereum/rpc` server on `ListenAndServe`, default port **8080** (`-l`). |
| RPC namespace | `main.go:105–107` | `RegisterName("swap", ss)` → methods are exposed as **`swap_*`** on the default HTTP handler (`/`). |
| Public methods | `swapService` | **`Swap`**, **`Forward`**, **`Backward`** (Go exported names; JSON-RPC clients typically call e.g. `swap_Swap` depending on client — verify with your RPC client). |

### Taker → node 0 (`Swap`)

```173:184:research/coinswapd/main.go
func (s *swapService) Swap(onion onion.Onion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.nodeIndex != 0 {
		return errors.New("node index is not zero")
	}
	if err := validateOnion(&onion); err != nil {
		return err
	}
	return saveOnion(db, &onion)
}
```

- Only **swap server (node index 0)** accepts new onions.
- Payload type is **`onion.Onion`** (see §2), not a custom “swap” wrapper.

### MLN `mln-cli` sidecar extension (not in upstream ltcmweb)

[`PHASE_10_TAKER_CLI.md`](../PHASE_10_TAKER_CLI.md) describes the taker CLI’s **optional** HTTP POST to an MLN-defined URL (default path **`/v1/swap`**) with a **route JSON** body (`tor` + `feeMinSat` per hop, plus `destination` and `amount` in satoshis). That keeps `mln-cli` as a **pure Go** coordinator while MWEB onion build and Tor dialing stay in **`coinswapd`**.

**Upstream `coinswapd` does not implement this endpoint**; it only accepts a full onion via JSON-RPC `swap_Swap` as above. A **fork or small proxy** next to `coinswapd` must translate the route JSON into `onion.Onion` (or otherwise drive the existing swap path).

**Normative fork blueprint** (JSON-RPC `mweb_submitRoute` / `mweb_getBalance`, key binding, milestones): [`research/COINSWAPD_MLN_FORK_SPEC.md`](COINSWAPD_MLN_FORK_SPEC.md).

For **wallet UX**, the same MLN extension may also expose **`GET /v1/balance`** (MWEB available/spendable satoshis for the taker wallet); see [`PHASE_10_TAKER_CLI.md`](../PHASE_10_TAKER_CLI.md) and [`mln-cli/internal/forger/balance.go`](../mln-cli/internal/forger/balance.go).

### Inter-node (`swap_forward` / `swap_backward`)

These are invoked via `rpc.Client.Call(nil, "swap_forward", data)` / `"swap_backward"` (`swap.go`) — i.e. **opaque binary blobs** after decryption, not JSON onions. The **forward** path sends **gob-encoded** sorted commitments + per-commitment `onionEtc`; the **backward** path sends outputs + kernels.

For LitVM **`evidenceHash` / `forwardCiphertextHash`** correlators (exact bytes to hash, `peeledCommitment` from `commit2`), see **`research/EVIDENCE_GENERATOR.md`**.

---

## 2. Taker onion: exact JSON-oriented shape

Defined in `onion/onion.go` with `encoding/json` tags (hex strings for byte fields):

| JSON field | Go field | Role |
|------------|----------|------|
| `input.output_id` | `OutputId` | UTXO id (hex) |
| `input.output_commit` | `Commitment` | Input commitment |
| `input.output_pk` | `OutputPubKey` | Receiver pk |
| `input.input_pk` | `InputPubKey` | Spend path |
| `input.input_sig` | `Signature` | MWEB input signature |
| `enc_payloads` | `Payloads` | Nested encrypted hop payloads |
| `ephemeral_xpub` | `PubKey` | X25519 ephemeral public key for this layer |
| `owner_proof` | `OwnerProof` | Extra Schnorr-style proof over the onion body |

Encryption matches the Grin/mwixnet README spirit: **ChaCha20** with key `HMAC-SHA256("MWIXNET", ecdh_secret)` and nonce **`NONCE1234567`** (`NewCipher`).

Each **hop** peel exposes a `Hop`:

- `KernelBlind`, `StealthBlind` (32-byte scalars),
- `Fee` (uint64),
- optional **`Output`** (`*wire.MwebOutput`) — required on the **last** node only (`peelOnions` enforces `lastNode == hasOutput`).

---

## 3. Cryptographic primitives (where to trace)

| Concern | Where |
|---------|--------|
| Onion build / peel | `onion/onion.go` — `New`, `Peel`, layered ChaCha XOR |
| Owner proof on onion | `Sign` / `VerifySig` — `mw.Sign`, blake3 key hash |
| Input validity | `validateOnion` — `cs.MwebCoinDB.FetchCoin`, `input.VerifySig()`, commitment vs chain |
| Commitment update after peel | `swap.go:71–72` — `commit2 := commit.Add(NewCommitment(&hop.KernelBlind,0)).Sub(NewCommitment({}, hop.Fee))` |
| Range proof (last hop) | `peelOnions` — `hop.Output.RangeProof.Verify(*commit2, msg)` |
| Tx assembly | `finalize` — `wire.MwebTxBody`, `cs.SendTransaction` |
| Fee output + kernel | `backward` — `mweb.CreateOutput`, `mweb.SignOutput`, `mweb.CreateKernel`, `mw.BlindSwitch` |

**No `mwebd` package** in `go.mod`; use **`ltcd`** as the library boundary for deeper MW math.

---

## 4. Hop routing, sorting, and “epochs”

| Topic | Implementation |
|-------|------------------|
| **Multi-hop forward** | `forward` → `peelOnions` → sort **commitment keys** with `big.Int` byte order → `gob` encode → XOR stream with shared key to **next** node → async `swap_forward`. |
| **Uniqueness** | If peeled `commit2` collides with an existing key, onion dropped. |
| **Backward** | Last node starts `backward`; each node appends its **fee output** + **kernel**, sorts outputs by `Hash()`, then XORs to **previous** node via `swap_backward`. |
| **Invariants** | `Backward` recomputes sums and checks `commitSum` vs `kernelExcess` and stealth sums (`errors` if mismatch). |

### Epoch / batching (not message-count based)

From `main.go` main loop:

- **Swap execution:** once per day when local time crosses **23:00 → 00:00** (`tPrev.Hour() == 23 && t.Hour() == 0`).
- **Refresh node list:** **00:00 → 01:00** (`getNodes`).

So the “anonymity set” is **whatever valid onions are persisted in the DB at midnight**, not a configurable min batch size in code.

---

## 5. Native MWEB fee mechanics

- Each hop carries **`hop.Fee`** in the peeled payload.
- In **`backward`**, the node aggregates `nodeFee` from all its onions’ hops and compares to a **derived per-node share** of the standard MWEB fee formula:

```206:215:research/coinswapd/swap.go
	nOutputs := len(outputs) + s.nodeIndex + 1
	nNodes := uint64(len(s.nodes))
	fee := uint64(nOutputs) * mweb.StandardOutputWeight * mweb.BaseMwebFee
	fee = (fee + nNodes - 1) / nNodes
	fee += mweb.KernelWithStealthWeight * mweb.BaseMwebFee

	if nodeFee < fee {
		return errors.New("insufficient hop fees")
	}
	nodeFee -= fee
```

- The node’s **share** of fees is realized as an **MWEB output** to **`-a` / `feeAddress`** (MWEB stealth address from flags), plus **`mweb.CreateKernel`** tying excess to that output (`kernelBlind`, `stealthBlind` updated with blind switch / sender key).

So compensation is **on-chain in the aggregated MWEB transaction**, not a separate LitVM rail.

---

## 6. Suggested reading order (matches your plan)

1. **`onion/onion.go`** — payload layout + `Hop` ↔ encryption (smallest file, defines the taker-visible JSON).
2. **`main.go`** — `Swap` + `validateOnion` + neutrino + **schedule**.
3. **`swap.go`** — `peelOnions`, `forward`, `backward`, `finalize` (full pipeline).

---

## 7. MLN stack notes

- **LitVM / Nostr / Tor** are **not** in this repo; staking and discovery are out of scope here.
- **Phase 0 gap analysis:** compare this onion + fee path to your MLN spec (`PRODUCT_SPEC.md`) for escrow/slashing **on top**, not as a replacement for Tromp/Hector MWEB fee math unless you deliberately fork economics.

---

## 8. Final transaction: `forward` → `backward` → `finalize`

This is the **# final transaction** path: how peeled onions become one broadcast **MWEB tx**.

### 8.1 Where the user outputs come from

On the **last** forward hop, `forward()` sees `nodeIndex == len(s.nodes)-1` and calls **`backward(outputs, nil)`** without going through another `swap_forward`. The **`outputs`** slice was filled in **`peelOnions`**: for each onion, the last peel must include a **`hop.Output`** (`*wire.MwebOutput`) that matches the updated commitment and stealth keys, with **range proof** and **output signature** verified (`swap.go` ~64–105).

So the **taker-supplied** destination outputs are already in `outputs` when the backward pass starts at the **final** node.

### 8.2 What one node does in `backward(outputs, kernels)`

1. **Re-peel** each onion (same hop as in `peelOnions`) to accumulate:
   - `kernelBlind`, `stealthBlind` (sums of hop scalars),
   - `nodeFee` (sum of `hop.Fee` for this node’s layer across all swaps).

2. **Compare** aggregated `nodeFee` to the **minimum** this node is allowed to take — a function of output count, node count, and MWEB weight constants (`StandardOutputWeight`, `BaseMwebFee`, `KernelWithStealthWeight`). If `nodeFee < fee`, the round **errors** (`insufficient hop fees`).

3. **Subtract** the protocol share: `nodeFee -= fee`. The remainder is value paid to this operator’s **`feeAddress`**:

   - Random `senderKey`, `mweb.CreateOutput` → **fee output** to stealth `feeAddress`,
   - `mweb.SignOutput`, update blinds with `mw.BlindSwitch` and sender key,
   - append that output and **`mweb.CreateKernel(kernelBlind, stealthBlind, &fee, …)`** so the kernel excess matches the updated aggregate blinds.

4. **Sort** all **`outputs`** by `Hash()` (lexicographic big-int order, same style as forward’s commitment sort).

5. **If `nodeIndex == 0`:** call **`finalize(outputs, kernels)`**.  
   **Else:** serialize **commitment keys still in `s.onions`**, output count, raw **outputs** and **kernels**, XOR with the **previous** node’s shared cipher, RPC **`swap_backward`** to the **previous** node.

### 8.3 What `Backward` does (middle / first nodes)

When a node receives **`swap_backward`**, it decrypts and decodes: list of **commits**, **output count**, then deserializes that many **`wire.MwebOutput`**, then deserializes **kernels from all downstream nodes**. It rebuilds **commitment and stealth sums** from outputs and kernels (subtracting fee terms from kernel excess), then **reconciles** with its local peeled onions: for each local commitment, if the derived `commit2` is in the decoded commit list, it subtracts that contribution from the sums; otherwise it **drops** the onion. If **commit** and **stealth** invariants fail, it errors; if OK, it calls **`backward`** again to prepend **its** fee output + kernel and either finalize (if index 0) or keep sending backward.

This implements Tromp’s **backward kernel pass**: each node adds its kernel leg so that aggregates line up with the sorted output set.

### 8.4 `finalize` — the actual broadcast

```365:383:research/coinswapd/swap.go
func (s *swapService) finalize(
	outputs []*wire.MwebOutput,
	kernels []*wire.MwebKernel) error {

	txBody := &wire.MwebTxBody{
		Outputs: outputs,
		Kernels: kernels,
	}
	for _, o := range s.onions {
		input, _ := inputFromOnion(o.Onion)
		txBody.Inputs = append(txBody.Inputs, input)
	}
	txBody.Sort()

	return cs.SendTransaction(&wire.MsgTx{
		Version: 2,
		Mweb:    &wire.MwebTx{TxBody: txBody},
	})
}
```

**Node 0** (swap server) attaches **every surviving input** from the stored onions (the original MWEB inputs validated at `Swap` time), **`Sort()`** the body, and sends **`MsgTx` v2** with **`Mweb`** payload via **`cs.SendTransaction`** (Neutrino `ChainService`).

### 8.5 Takeaway for MLN

- **Final tx** = one **aggregated MWEB transaction**: user outputs + per-node fee outputs + kernels + all inputs.  
- **LitVM** does not participate in building this tx in `coinswapd`; any **staking/slashing** contract is an **orthogonal** layer (deposit/bounty only), consistent with **hybrid** economics in `PRODUCT_SPEC.md` §5.2.
