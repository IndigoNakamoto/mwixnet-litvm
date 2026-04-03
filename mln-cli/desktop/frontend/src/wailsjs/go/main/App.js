// Hand-maintained bridge for Wails bindings (regenerate with `wails generate module` when API changes).

const isDevBrowser =
  typeof import.meta !== 'undefined' &&
  import.meta.env?.DEV &&
  typeof window !== 'undefined' &&
  !window.go?.main?.App

function app() {
  const a = window.go?.main?.App
  if (!a) {
    if (isDevBrowser) {
      return null
    }
    throw new Error('Wails runtime not available (run with wails dev or build the desktop app).')
  }
  return a
}

const mockSettings = () =>
  Promise.resolve({
    nostrRelays: [],
    litvmChainId: '',
    litvmHttpUrl: '',
    registryAddr: '',
    grievanceCourtAddr: '',
    scoutTimeout: '30s',
    defaultSidecarUrl: 'http://127.0.0.1:8080/v1/swap',
    forgerHttpTimeout: '10s',
    selfIncludedRouting: false,
    operatorEthPrivateKeyHex: '',
  })

export function LoadSettings() {
  const a = app()
  if (!a) return mockSettings()
  return a.LoadSettings()
}

export function SaveSettings(s) {
  const a = app()
  if (!a) {
    console.warn('[dev] SaveSettings (mock):', s)
    return Promise.resolve()
  }
  return a.SaveSettings(s)
}

export function Scout() {
  const a = app()
  if (!a) {
    return Promise.resolve({
      verified: [],
      rejected: [],
    })
  }
  return a.Scout()
}

export function BuildRoute() {
  const a = app()
  if (!a) {
    return Promise.reject(
      new Error('[dev mock] Build route needs the real app (3+ makers on network). Run mln-wallet or wails dev.')
    )
  }
  return a.BuildRoute()
}

export function DryRunRouteJSON(routeJSON) {
  const a = app()
  if (!a) {
    try {
      const o = JSON.parse(routeJSON)
      const hops = o.hops || []
      return Promise.resolve({
        hops: hops.map((h, i) => ({
          index: i + 1,
          tor: h.tor || h.Tor || '',
        })),
      })
    } catch {
      return Promise.reject(new Error('invalid route JSON'))
    }
  }
  return a.DryRunRouteJSON(routeJSON)
}

export function Send(routeJSON, dest, amountSat, sidecarURL) {
  const a = app()
  if (!a) {
    return Promise.reject(
      new Error('[dev mock] Send requires coinswapd sidecar. Use the desktop build (wails / go run -tags=wails).')
    )
  }
  return a.Send(routeJSON, dest, amountSat, sidecarURL)
}

export function FetchMwebBalance(sidecarURL) {
  const a = app()
  if (!a) {
    return Promise.resolve({
      availableSat: 125_000_000,
      spendableSat: 125_000_000,
      detail: '[dev mock] 1.25 LTC in MWEB',
    })
  }
  return a.FetchMwebBalance(sidecarURL)
}
