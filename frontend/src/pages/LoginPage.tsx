import { useState, type FormEvent } from 'react'
import { ArrowRight, BadgeCheck, Eye, EyeOff, Fingerprint, LockKeyhole, ShieldCheck } from 'lucide-react'
import { Logo } from '../components/Logo'
import { useAuth } from '../state/AuthContext'

export function LoginPage() {
  const [kgid, setKgid] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const { login, loginWithSSO, ssoEnabled } = useAuth()

  async function submit(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError('')
    try { await login(kgid.trim(), password) } catch (err) { setError(err instanceof Error ? err.message : 'Unable to sign in') } finally { setBusy(false) }
  }

  return (
    <main className="login-page">
      <section className="login-context">
        <div className="login-brand"><Logo /></div>
        <div className="context-copy">
          <span className="eyebrow"><i /> SECURE · INTELLIGENT · ACCOUNTABLE</span>
          <h1>One operational picture.<br /><em>Better investigation.</em></h1>
          <p>Turn FIR records, field evidence and crime patterns into trusted, actionable intelligence—within your jurisdiction.</p>
          <div className="trust-row"><span><Fingerprint /> Scoped access</span><span><BadgeCheck /> Evidence grounded</span><span><ShieldCheck /> Audit protected</span></div>
        </div>
        <div className="data-visual" aria-hidden="true"><i /><i /><i /><i /><i /><span /><span /><span /></div>
        <footer>Authorized law-enforcement use only · Karnataka State Police</footer>
      </section>
      <section className="login-form-side">
        <form className="login-card glass-panel" onSubmit={submit}>
          <div className="login-card-icon"><LockKeyhole size={23} /></div>
          <span className="form-eyebrow">OFFICER ACCESS</span>
          <h2>Welcome back</h2>
          <p>Sign in with your department credentials.</p>
          {error && <div className="form-error">{error}</div>}
          <label>KGID / Officer ID<input autoFocus autoComplete="username" required value={kgid} onChange={(e) => setKgid(e.target.value)} placeholder="Enter your KGID" /></label>
          <label>Password<div className="password-field"><input type={showPassword ? 'text' : 'password'} autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Enter your password" /><button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Hide password' : 'Show password'}>{showPassword ? <EyeOff size={18} /> : <Eye size={18} />}</button></div></label>
          <div className="login-options"><label className="check-label"><input type="checkbox" /> Keep me signed in</label><button type="button" className="text-button">Need access help?</button></div>
          <button className="button primary login-submit" disabled={busy}>{busy ? 'Verifying credentials…' : 'Secure sign in'}<ArrowRight size={18} /></button>
          {ssoEnabled && <><div className="login-divider"><span>OR</span></div><button type="button" className="button secondary login-submit" onClick={loginWithSSO}><ShieldCheck size={17} /> Department SSO<ArrowRight size={17} /></button></>}
          <div className="login-security"><ShieldCheck size={16} /><span>Your session is encrypted, access-controlled and logged.</span></div>
        </form>
        <small className="login-help">For access issues, contact your district system administrator.</small>
      </section>
    </main>
  )
}
