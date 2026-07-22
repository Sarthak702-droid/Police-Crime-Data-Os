const issuer = (import.meta.env.VITE_OIDC_ISSUER || '').replace(/\/$/, '')
const clientID = import.meta.env.VITE_OIDC_CLIENT_ID || 'crime-api'
const callbackURL = () => `${location.origin}/`

function base64url(bytes: Uint8Array) {
  return btoa(String.fromCharCode(...bytes)).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '')
}

export const oidcEnabled = () => import.meta.env.VITE_AUTH_MODE === 'oidc' && Boolean(issuer)

export async function startOIDCLogin() {
  const verifier = base64url(crypto.getRandomValues(new Uint8Array(48)))
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier))
  const state = base64url(crypto.getRandomValues(new Uint8Array(24)))
  sessionStorage.setItem('drishti_pkce_verifier', verifier)
  sessionStorage.setItem('drishti_oidc_state', state)
  const params = new URLSearchParams({ client_id: clientID, redirect_uri: callbackURL(), response_type: 'code', scope: 'openid profile', state, code_challenge: base64url(new Uint8Array(digest)), code_challenge_method: 'S256' })
  location.assign(`${issuer}/protocol/openid-connect/auth?${params}`)
}

async function tokenRequest(params: URLSearchParams) {
  const response = await fetch(`${issuer}/protocol/openid-connect/token`, { method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' }, body: params })
  if (!response.ok) throw new Error('Department identity sign-in failed')
  const tokens = await response.json() as { access_token: string; refresh_token?: string }
  localStorage.setItem('drishti_access_token', tokens.access_token)
  if (tokens.refresh_token) localStorage.setItem('drishti_refresh_token', tokens.refresh_token)
}

export async function completeOIDCCallback() {
  const params = new URLSearchParams(location.search)
  const code = params.get('code'); if (!code) return false
  if (params.get('state') !== sessionStorage.getItem('drishti_oidc_state')) throw new Error('Invalid identity callback state')
  const verifier = sessionStorage.getItem('drishti_pkce_verifier') || ''
  await tokenRequest(new URLSearchParams({ grant_type: 'authorization_code', client_id: clientID, code, redirect_uri: callbackURL(), code_verifier: verifier }))
  sessionStorage.removeItem('drishti_pkce_verifier'); sessionStorage.removeItem('drishti_oidc_state')
  history.replaceState({}, document.title, location.pathname)
  return true
}

export async function refreshOIDCToken(refreshToken: string) {
  if (!oidcEnabled()) return false
  try { await tokenRequest(new URLSearchParams({ grant_type: 'refresh_token', client_id: clientID, refresh_token: refreshToken })); return true } catch { return false }
}
