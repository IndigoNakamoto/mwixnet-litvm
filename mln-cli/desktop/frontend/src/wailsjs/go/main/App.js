// Hand-maintained bridge for Wails bindings (regenerate with `wails generate module` when API changes).

function app() {
  const a = window.go?.main?.App
  if (!a) {
    throw new Error('Wails runtime not available (run with wails dev or build the desktop app).')
  }
  return a
}

export function LoadSettings() {
  return app().LoadSettings()
}

export function SaveSettings(s) {
  return app().SaveSettings(s)
}

export function Scout() {
  return app().Scout()
}

export function BuildRoute() {
  return app().BuildRoute()
}

export function DryRunRouteJSON(routeJSON) {
  return app().DryRunRouteJSON(routeJSON)
}

export function Send(routeJSON, dest, amountSat, sidecarURL) {
  return app().Send(routeJSON, dest, amountSat, sidecarURL)
}
