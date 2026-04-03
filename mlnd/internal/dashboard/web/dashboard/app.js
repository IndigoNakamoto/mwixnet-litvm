/* global fetch, EventSource */

const token = new URLSearchParams(window.location.search).get("token");
const headers = {};
if (token) {
  headers["X-MLND-Token"] = token;
}

function authFetch(url, opts = {}) {
  return fetch(url, { ...opts, headers: { ...headers, ...opts.headers } });
}

function setConn(cls, text) {
  const el = document.getElementById("conn");
  el.className = "pill " + cls;
  el.textContent = text;
}

/** Truncate 0x hex for display; keep full value in data-full for copy. */
function truncateHexMiddle(s, leftInner, rightInner) {
  if (!s || s.length < 12) return s || "";
  if (!s.startsWith("0x")) return s;
  const body = s.slice(2);
  if (body.length <= leftInner + rightInner) return s;
  return "0x" + body.slice(0, leftInner) + "…" + body.slice(-rightInner);
}

function setHexField(id, full) {
  const el = document.getElementById(id);
  if (!full) {
    el.textContent = "—";
    el.removeAttribute("data-full");
    return;
  }
  el.dataset.full = full;
  el.textContent = truncateHexMiddle(full, 10, 8);
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const ta = document.createElement("textarea");
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  }
}

document.addEventListener("click", (e) => {
  const btn = e.target.closest(".js-copy");
  if (!btn) return;
  const id = btn.getAttribute("data-for");
  const el = document.getElementById(id);
  const full = el && el.dataset.full;
  if (full) {
    copyText(full);
    const prev = btn.textContent;
    btn.textContent = "Copied";
    setTimeout(() => {
      btn.textContent = prev;
    }, 1200);
  }
});

function humanDuration(seconds) {
  const s = Number(seconds);
  if (!Number.isFinite(s) || s < 0) return "";
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m`;
  if (s < 86400) return `${Math.floor(s / 3600)}h`;
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  return h > 0 ? `${d}d ${h}h` : `${d}d`;
}

function fmtExit(unlock, cooldown) {
  const u = BigInt(unlock || "0");
  if (u === 0n) return "Not in exit queue";
  const now = BigInt(Math.floor(Date.now() / 1000));
  const cd = cooldown ? BigInt(cooldown) : 0n;
  if (now >= u) {
    return `Unlock time reached (${u}) — confirm withdrawal lock and open grievances before withdrawing.`;
  }
  const left = u - now;
  const eta = new Date(Number(u) * 1000).toISOString();
  return `~${humanDuration(Number(left))} remaining (until ${eta} UTC) · cooldown parameter ${cd}s`;
}

function fmtWithdrawalLock(ts) {
  if (!ts || ts === "0") return "—";
  const n = BigInt(ts);
  if (n === 0n) return "—";
  const now = BigInt(Math.floor(Date.now() / 1000));
  const iso = new Date(Number(n) * 1000).toISOString();
  if (now >= n) return `Past (${iso} UTC)`;
  return `${iso} UTC (~${humanDuration(Number(n - now))})`;
}

function apiURL(path) {
  if (!token) return path;
  const sep = path.includes("?") ? "&" : "?";
  return path + sep + "token=" + encodeURIComponent(token);
}

function showEl(id, show) {
  const el = document.getElementById(id);
  if (!el) return;
  el.classList.toggle("hidden", !show);
}

async function loadStatus() {
  const r = await authFetch(apiURL("/api/v1/status"));
  if (!r.ok) {
    setConn("err", "API " + r.status);
    if (r.status === 401) {
      showEl("banner-auth", true);
      const b = document.getElementById("banner-auth");
      b.textContent =
        "Unauthorized. If MLND_HTTP_TOKEN is set, open this page with ?token=YOUR_TOKEN in the URL (EventSource cannot send custom headers).";
    } else {
      showEl("banner-auth", false);
    }
    return;
  }
  showEl("banner-auth", false);

  const s = await r.json();
  setConn("ok", "Live");

  const rpcHint = document.getElementById("rpc-hint");
  if (s.connection && s.connection.litvmRpcLabel) {
    rpcHint.textContent = "RPC " + s.connection.litvmRpcLabel;
  } else {
    rpcHint.textContent = "";
  }

  const g = s.grievanceNarrative;
  if (g && g.headline) {
    showEl("grievance-strip", true);
    const strip = document.getElementById("grievance-strip");
    strip.classList.remove("level-info", "level-warn", "level-err", "level-critical");
    const lv = (g.level || "info").toLowerCase();
    if (lv === "critical") strip.classList.add("level-critical");
    else if (lv === "error") strip.classList.add("level-err");
    else if (lv === "warn") strip.classList.add("level-warn");
    else strip.classList.add("level-info");
    document.getElementById("grievance-headline").textContent = g.headline;
    let det = g.detail || "";
    if (g.data && g.data.grievanceId) det += (det ? " · " : "") + "Case " + truncateHexMiddle(g.data.grievanceId, 8, 6);
    if (g.data && g.data.deadline) det += (det ? " · " : "") + "Deadline (chain): " + g.data.deadline;
    document.getElementById("grievance-detail").textContent = det;
  } else {
    showEl("grievance-strip", false);
  }

  const c = s.chain || {};
  const rpcErr = c.rpcError;
  if (rpcErr) {
    showEl("banner-rpc", true);
    document.getElementById("chain-overlay-detail").textContent = rpcErr;
    showEl("chain-degraded-overlay", true);
    document.getElementById("chain-grid").classList.add("faded");
  } else {
    showEl("banner-rpc", false);
    showEl("chain-degraded-overlay", false);
    document.getElementById("chain-grid").classList.remove("faded");
  }

  setHexField("op", c.operator);
  document.getElementById("stake").textContent = rpcErr ? "—" : c.stake || "—";
  document.getElementById("frozen").textContent = rpcErr ? "—" : String(c.stakeFrozen);
  document.getElementById("gcount").textContent = rpcErr ? "—" : c.openGrievanceCount ?? "—";
  document.getElementById("exitunlock").textContent = rpcErr ? "—" : fmtExit(c.exitUnlockTime, c.cooldownPeriodSeconds);
  document.getElementById("wdlock").textContent = rpcErr ? "—" : fmtWithdrawalLock(c.withdrawalLockUntil);
  if (c.courtAddressMismatch && !rpcErr) {
    document.getElementById("frozen").textContent += " (court env ≠ registry.grievanceCourt)";
  }

  const b = s.daemon || {};
  const ban = document.getElementById("banner-auto");
  if (!b.autoDefend) {
    ban.textContent =
      b.autoDefendWarning ||
      "Auto-defend is OFF — if a grievance is filed, submit defense manually before the deadline or risk slashing.";
    ban.classList.remove("hidden");
  } else {
    ban.classList.add("hidden");
  }

  const n = s.network || {};
  setHexField("dtag", n.dTag);

  const warm = document.getElementById("nostr-warm");
  let rs = "";
  if (n.error) {
    warm.classList.remove("hidden");
    warm.textContent =
      "Discovery check: " +
      n.error +
      " — without a visible maker ad, takers using these relays may not route to you.";
    rs = n.error;
  } else if (n.eventFound) {
    warm.classList.add("hidden");
    rs =
      "Ad found via " +
      (n.relayQueried || "relay") +
      " · event " +
      truncateHexMiddle(n.eventId || "", 8, 6) +
      "\nRegistry binding: " +
      (n.nostrKeyHashMatch ? "ok" : "mismatch") +
      " · Stake rule: " +
      (n.registryOK ? "ok" : "fail");
    if (n.verifyReason) rs += " (" + n.verifyReason + ")";
  } else {
    warm.classList.remove("hidden");
    warm.textContent = "Waiting for relay query…";
    rs = "—";
  }
  document.getElementById("relay-status").textContent = rs;

  let drift = "—";
  if (n.localSwapX25519Expected && n.eventFound) {
    drift = n.swapKeyDrift
      ? "Mismatch: relay ad key differs from MLND_SWAP_X25519_PUB_HEX — fix env and restart mlnd so the next publish matches."
      : "Relay ad matches configured swap key.";
  } else if (n.localSwapX25519Expected) {
    drift = "Key set in env; no matching relay event yet (publish requires MLND_NOSTR_NSEC).";
  } else {
    drift = "Optional: set MLND_SWAP_X25519_PUB_HEX when your engine exposes a stable X25519 pubkey.";
  }
  document.getElementById("swap-drift").textContent = drift;

  document.getElementById("lastpub").textContent =
    n.lastPublish && n.lastPublish.relays && n.lastPublish.relays.length
      ? JSON.stringify(n.lastPublish, null, 2)
      : "—";

  let intended = "—";
  if (n.intendedAdContent) {
    try {
      intended = JSON.stringify(JSON.parse(n.intendedAdContent), null, 2);
    } catch {
      intended = n.intendedAdContent;
    }
  }
  document.getElementById("intended").textContent = intended;

  const e = s.engine || {};
  document.getElementById("receipts").textContent = e.receiptCountOk
    ? String(e.receiptCount) + " row(s)"
    : "Could not count: " + (e.receiptErr || "?");
  document.getElementById("bridge").textContent = e.bridgeEnabled
    ? "Watching " + (e.bridgeDir || "(dir)")
    : "Off — set MLND_BRIDGE_COINSWAPD=1 and MLND_BRIDGE_RECEIPTS_DIR to ingest hop receipts.";

  const mwebWarm = document.getElementById("mweb-warm");
  if (e.bridgeEnabled) {
    mwebWarm.textContent = "Bridge is polling for new NDJSON lines into the receipt vault.";
  } else {
    mwebWarm.textContent =
      "Receipt vault is local SQLite only until the coinswapd bridge is enabled (see mlnd README).";
  }
}

function startSSE() {
  const es = new EventSource(apiURL("/api/v1/events"));
  const box = document.getElementById("events");
  es.onmessage = (ev) => {
    try {
      const j = JSON.parse(ev.data);
      const line =
        new Date((j.ts || 0) * 1000).toISOString() +
        " [" +
        j.level +
        "] " +
        j.code +
        " — " +
        j.message +
        "\n";
      box.textContent = line + box.textContent;
    } catch {
      box.textContent = ev.data + "\n" + box.textContent;
    }
  };
  es.onerror = () => {};
}

loadStatus().catch(() => setConn("err", "Could not reach API"));
setInterval(() => loadStatus().catch(() => {}), 15000);
startSSE();
