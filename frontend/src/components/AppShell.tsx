import { useEffect, useState, type ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Bell, Bot, ChevronDown, FilePlus2, FolderSearch2, LayoutDashboard, LogOut, Menu, Network, Search, Shield, X } from 'lucide-react'
import { Logo } from './Logo'
import { useAuth } from '../state/AuthContext'

const nav = [
  { to: '/', label: 'Command dashboard', icon: LayoutDashboard },
  { to: '/cases', label: 'Case registry', icon: FolderSearch2 },
  { to: '/cases/new', label: 'Register FIR', icon: FilePlus2 },
  { to: '/copilot', label: 'AI investigation copilot', icon: Bot },
  { to: '/intelligence', label: 'Crime intelligence', icon: Network },
]

const titles: Record<string, [string, string]> = {
  '/': ['Command dashboard', 'Station-wide operational picture'],
  '/cases': ['Case registry', 'Search and manage investigation records'],
  '/cases/new': ['Register new FIR', 'Structured first information report'],
  '/copilot': ['AI investigation copilot', 'Grounded answers from authorized police records'],
  '/intelligence': ['Crime intelligence', 'Patterns, priorities and criminal networks'],
}

export function AppShell({ children }: { children: ReactNode }) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [profileOpen, setProfileOpen] = useState(false)
  const [online, setOnline] = useState(navigator.onLine)
  const location = useLocation()
  const { officer, logout } = useAuth()
  useEffect(() => {
    const update = () => setOnline(navigator.onLine)
    addEventListener('online', update); addEventListener('offline', update)
    return () => { removeEventListener('online', update); removeEventListener('offline', update) }
  }, [])
  useEffect(() => setMobileOpen(false), [location.pathname])
  const exact = titles[location.pathname]
  const page = exact || (location.pathname.startsWith('/cases/') ? ['Case workspace', 'Investigation record and decision support'] : titles['/'])
  const name = officer?.FirstName || officer?.first_name || officer?.KGID || officer?.kgid || 'Officer'
  const rank = officer?.rank || officer?.designation || 'Investigating officer'
  const initials = String(name).split(' ').map((part) => part[0]).join('').slice(0, 2).toUpperCase()

  return (
    <div className="app-shell">
      {mobileOpen && <button className="nav-scrim" aria-label="Close navigation" onClick={() => setMobileOpen(false)} />}
      <aside className={`sidebar ${mobileOpen ? 'open' : ''}`}>
        <div className="sidebar-head"><Logo /><button className="icon-button mobile-only" onClick={() => setMobileOpen(false)}><X size={20} /></button></div>
        <div className="station-card">
          <span className="station-icon"><Shield size={17} /></span>
          <div><small>CURRENT JURISDICTION</small><strong>Unit {officer?.UnitID || officer?.unit_id || '—'}</strong><span>District {officer?.DistrictID || officer?.district_id || '—'}</span></div>
        </div>
        <nav className="main-nav" aria-label="Primary navigation">
          <span className="nav-section">OPERATIONS</span>
          {nav.slice(0, 3).map(({ to, label, icon: Icon }) => <NavLink key={to} to={to} end={to === '/'}><Icon size={19} /><span>{label}</span></NavLink>)}
          <span className="nav-section">INTELLIGENCE</span>
          {nav.slice(3).map(({ to, label, icon: Icon }) => <NavLink key={to} to={to}><Icon size={19} /><span>{label}</span>{to === '/copilot' && <i>AI</i>}</NavLink>)}
        </nav>
        <div className="sidebar-foot">
          <div className="security-note"><ShieldCheckIcon /><div><strong>Secure workspace</strong><span>Activity is logged and audited</span></div></div>
          <span className="system-state"><i className={online ? 'online' : ''} />{online ? 'Systems operational' : 'Network unavailable'}</span>
        </div>
      </aside>
      <div className="app-main">
        <header className="topbar">
          <button className="icon-button menu-button" onClick={() => setMobileOpen(true)}><Menu size={21} /></button>
          <div className="page-heading"><h1>{page[0]}</h1><p>{page[1]}</p></div>
          <div className="top-actions">
            <button className="top-search" onClick={() => document.getElementById('global-search')?.focus()}><Search size={17} /><span>Search records</span><kbd>⌘ K</kbd></button>
            <button className="icon-button notification"><Bell size={19} /><i /></button>
            <div className="profile-wrap">
              <button className="profile-button" onClick={() => setProfileOpen(!profileOpen)}><span className="avatar">{initials}</span><span><strong>{name}</strong><small>{rank}</small></span><ChevronDown size={15} /></button>
              {profileOpen && <div className="profile-menu glass-panel"><button onClick={logout}><LogOut size={16} /> Sign out securely</button></div>}
            </div>
          </div>
        </header>
        <main className="content">{children}</main>
      </div>
    </div>
  )
}

function ShieldCheckIcon() { return <div className="security-pulse"><Shield size={16} /></div> }
