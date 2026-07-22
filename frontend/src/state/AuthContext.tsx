import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { get, post } from '../lib/api'
import type { Officer } from '../types'
import { completeOIDCCallback, oidcEnabled, startOIDCLogin } from '../lib/oidc'

interface AuthState {
  officer: Officer | null
  loading: boolean
  login: (kgid: string, password: string) => Promise<void>
  loginWithSSO: () => Promise<void>
  ssoEnabled: boolean
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [officer, setOfficer] = useState<Officer | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const initialize = async () => {
      if (oidcEnabled() && location.search.includes('code=')) await completeOIDCCallback()
      if (!localStorage.getItem('drishti_access_token')) return
      setOfficer(await get<Officer>('/auth/me'))
    }
    initialize().catch(() => {
      localStorage.removeItem('drishti_access_token')
      localStorage.removeItem('drishti_refresh_token')
    }).finally(() => setLoading(false))
  }, [])

  async function login(kgid: string, password: string) {
    const result = await post<{ token: string; refresh_token: string; employee: Officer }>('/auth/login', { kgid, password })
    localStorage.setItem('drishti_access_token', result.token)
    localStorage.setItem('drishti_refresh_token', result.refresh_token)
    setOfficer(result.employee)
  }

  function logout() {
    localStorage.removeItem('drishti_access_token')
    localStorage.removeItem('drishti_refresh_token')
    setOfficer(null)
  }

  const value = useMemo(() => ({ officer, loading, login, logout, loginWithSSO: startOIDCLogin, ssoEnabled: oidcEnabled() }), [officer, loading])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}
