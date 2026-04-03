import { useCallback, useEffect, useMemo, useState } from 'react'
import * as AppGo from './wailsjs/go/main/App.js'

function relaysToText(relays) {
  if (!relays || !relays.length) return ''
  return relays.join(', ')
}

function emptySettings() {
  return {
    nostrRelays: [],
    litvmChainId: '',
    litvmHttpUrl: '',
    registryAddr: '',
    grievanceCourtAddr: '',
    scoutTimeout: '30s',
    defaultSidecarUrl: 'http://127.0.0.1:8080/v1/swap',
    forgerHttpTimeout: '10s',
  }
}

export default function App() {
  const [settings, setSettings] = useState(emptySettings)
  const [relayText, setRelayText] = useState('')
  const [preset, setPreset] = useState('privacy')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const [ok, setOk] = useState('')
  const [scoutSummary, setScoutSummary] = useState(null)
  const [routeJSON, setRouteJSON] = useState('')
  const [routeMeta, setRouteMeta] = useState(null)
  const [dryResult, setDryResult] = useState(null)
  const [dest, setDest] = useState('')
  const [amountSat, setAmountSat] = useState('')
  const [sidecarURL, setSidecarURL] = useState('')

  const load = useCallback(async () => {
    setErr('')
    setOk('')
    try {
      const s = await AppGo.LoadSettings()
      setSettings({ ...emptySettings(), ...s })
      setRelayText(relaysToText(s.nostrRelays))
      if (s.defaultSidecarUrl) setSidecarURL(s.defaultSidecarUrl)
    } catch (e) {
      setErr(String(e.message || e))
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const save = async () => {
    setBusy(true)
    setErr('')
    setOk('')
    try {
      const relays = relayText
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
      const payload = {
        ...settings,
        nostrRelays: relays,
      }
      await AppGo.SaveSettings(payload)
      setSettings(payload)
      setOk('Settings saved.')
    } catch (e) {
      setErr(String(e.message || e))
    } finally {
      setBusy(false)
    }
  }

  const runScout = async () => {
    setBusy(true)
    setErr('')
    setOk('')
    setScoutSummary(null)
    try {
      const r = await AppGo.Scout()
      setScoutSummary(r)
      setOk(`Scout: ${r.verified?.length ?? 0} verified makers.`)
    } catch (e) {
      setErr(String(e.message || e))
    } finally {
      setBusy(false)
    }
  }

  const runBuildRoute = async () => {
    setBusy(true)
    setErr('')
    setOk('')
    setRouteJSON('')
    setRouteMeta(null)
    setDryResult(null)
    try {
      const r = await AppGo.BuildRoute()
      const j = JSON.stringify(r.route, null, 2)
      setRouteJSON(j)
      setRouteMeta({
        verifiedCount: r.verifiedCount,
        rejectedCount: r.rejectedCount,
        feeSumSat: r.feeSumSat,
      })
      setOk('Route built. Review hops, then send.')
    } catch (e) {
      setErr(String(e.message || e))
    } finally {
      setBusy(false)
    }
  }

  const runDryRun = async () => {
    if (!routeJSON.trim()) {
      setErr('Build a route first.')
      return
    }
    setBusy(true)
    setErr('')
    setOk('')
    try {
      const d = await AppGo.DryRunRouteJSON(routeJSON)
      setDryResult(d)
      setOk('Dry-run: Tor endpoints OK.')
    } catch (e) {
      setErr(String(e.message || e))
    } finally {
      setBusy(false)
    }
  }

  const runSend = async () => {
    if (!routeJSON.trim()) {
      setErr('Build a route first.')
      return
    }
    const amt = Number(amountSat)
    if (!dest.trim()) {
      setErr('Destination MWEB address is required.')
      return
    }
    if (!Number.isFinite(amt) || amt <= 0) {
      setErr('Amount (satoshis) must be a positive number.')
      return
    }
    setBusy(true)
    setErr('')
    setOk('')
    try {
      const r = await AppGo.Send(routeJSON, dest.trim(), Math.floor(amt), sidecarURL.trim())
      const parts = [r.detail && `Detail: ${r.detail}`, r.epochNote].filter(Boolean)
      setOk(parts.join('\n\n'))
    } catch (e) {
      setErr(String(e.message || e))
    } finally {
      setBusy(false)
    }
  }

  const hopsPreview = useMemo(() => {
    if (!routeJSON.trim()) return null
    try {
      const o = JSON.parse(routeJSON)
      const hops = o.hops || o.Hops
      if (!hops) return null
      return hops
    } catch {
      return null
    }
  }, [routeJSON])

  return (
    <div>
      <h1>MLN Wallet</h1>
      <p className="sub">Taker flow: configure network → scout → build route → send via local coinswapd sidecar.</p>

      <section>
        <h2>Network settings</h2>
        <label htmlFor="relays">Nostr relays (comma-separated wss://)</label>
        <textarea
          id="relays"
          value={relayText}
          onChange={(e) => setRelayText(e.target.value)}
          placeholder="wss://relay.example.com"
        />
        <label htmlFor="chain">LitVM chain id (decimal)</label>
        <input
          id="chain"
          value={settings.litvmChainId}
          onChange={(e) => setSettings((s) => ({ ...s, litvmChainId: e.target.value }))}
        />
        <label htmlFor="rpc">LitVM HTTP JSON-RPC URL</label>
        <input
          id="rpc"
          value={settings.litvmHttpUrl}
          onChange={(e) => setSettings((s) => ({ ...s, litvmHttpUrl: e.target.value }))}
          placeholder="https://rpc.example"
        />
        <label htmlFor="reg">Registry address (0x…)</label>
        <input
          id="reg"
          value={settings.registryAddr}
          onChange={(e) => setSettings((s) => ({ ...s, registryAddr: e.target.value }))}
        />
        <label htmlFor="court">Grievance court (optional 0x…)</label>
        <input
          id="court"
          value={settings.grievanceCourtAddr}
          onChange={(e) => setSettings((s) => ({ ...s, grievanceCourtAddr: e.target.value }))}
        />
        <label htmlFor="scoutto">Scout timeout (e.g. 30s)</label>
        <input
          id="scoutto"
          value={settings.scoutTimeout}
          onChange={(e) => setSettings((s) => ({ ...s, scoutTimeout: e.target.value }))}
        />
        <label htmlFor="sidecardef">Default sidecar URL</label>
        <input
          id="sidecardef"
          value={settings.defaultSidecarUrl}
          onChange={(e) => {
            setSettings((s) => ({ ...s, defaultSidecarUrl: e.target.value }))
            setSidecarURL(e.target.value)
          }}
        />
        <label htmlFor="forgerto">Forger HTTP timeout (e.g. 10s)</label>
        <input
          id="forgerto"
          value={settings.forgerHttpTimeout}
          onChange={(e) => setSettings((s) => ({ ...s, forgerHttpTimeout: e.target.value }))}
        />
        <div className="row">
          <button type="button" className="primary" disabled={busy} onClick={save}>
            Save settings
          </button>
          <button type="button" disabled={busy} onClick={load}>
            Reload
          </button>
        </div>
      </section>

      <section>
        <h2>Route policy (PoC)</h2>
        <p className="sub" style={{ marginTop: 0 }}>
          Preset is stored for UX only; routing matches <span className="mono">mln-cli pathfind</span> today.
        </p>
        <div className="preset">
          <button type="button" className={preset === 'fast' ? 'primary' : ''} onClick={() => setPreset('fast')}>
            Fast
          </button>
          <button type="button" className={preset === 'privacy' ? 'primary' : ''} onClick={() => setPreset('privacy')}>
            Privacy
          </button>
        </div>
        <div className="row">
          <button type="button" disabled={busy} onClick={runScout}>
            Scout makers
          </button>
          <button type="button" className="primary" disabled={busy} onClick={runBuildRoute}>
            Build route
          </button>
        </div>
        {scoutSummary && (
          <p className="ok" style={{ marginTop: '0.75rem' }}>
            Verified: {scoutSummary.verified?.length ?? 0}, rejected events: {scoutSummary.rejected?.length ?? 0}
          </p>
        )}
        {routeMeta && (
          <p className="mono" style={{ marginTop: '0.5rem', color: 'var(--muted)' }}>
            fee_sum_sat_hint={routeMeta.feeSumSat} · verified={routeMeta.verifiedCount} · rejected=
            {routeMeta.rejectedCount}
          </p>
        )}
        {hopsPreview && (
          <div style={{ marginTop: '0.75rem' }}>
            {hopsPreview.map((h, i) => (
              <div key={i} className="hop">
                <strong>N{i + 1}</strong>{' '}
                <span className="mono">{h.operator || h.Operator}</span>
                <div className="mono" style={{ marginTop: 2 }}>
                  {h.tor || h.Tor}
                </div>
              </div>
            ))}
          </div>
        )}
        <div className="row" style={{ marginTop: '0.75rem' }}>
          <button type="button" disabled={busy || !routeJSON} onClick={runDryRun}>
            Dry-run Tor
          </button>
        </div>
        {dryResult?.hops && (
          <ul style={{ margin: '0.5rem 0 0', paddingLeft: '1.1rem', fontSize: '0.88rem' }}>
            {dryResult.hops.map((h) => (
              <li key={h.index}>
                N{h.index}: <span className="mono">{h.tor}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2>Send (sidecar)</h2>
        <label htmlFor="dest">Destination MWEB address</label>
        <input id="dest" value={dest} onChange={(e) => setDest(e.target.value)} placeholder="mweb1…" />
        <label htmlFor="amt">Amount (satoshis)</label>
        <input
          id="amt"
          type="number"
          min="1"
          step="1"
          value={amountSat}
          onChange={(e) => setAmountSat(e.target.value)}
        />
        <label htmlFor="sidecar">Sidecar URL (optional override)</label>
        <input
          id="sidecar"
          value={sidecarURL}
          onChange={(e) => setSidecarURL(e.target.value)}
          placeholder="http://127.0.0.1:8080/v1/swap"
        />
        <button type="button" className="primary" disabled={busy} onClick={runSend}>
          Send privately
        </button>
      </section>

      {err ? <p className="err">{err}</p> : null}
      {ok ? <p className="ok">{ok}</p> : null}
    </div>
  )
}
