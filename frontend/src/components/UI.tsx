import type { LucideIcon } from 'lucide-react'
import { AlertCircle, CheckCircle2, LoaderCircle, SearchX } from 'lucide-react'
import type { ReactNode } from 'react'

export function StatCard({ label, value, detail, icon: Icon, tone = 'teal' }: { label: string; value: string | number; detail: string; icon: LucideIcon; tone?: string }) {
  return <article className={`stat-card glass-panel ${tone}`}><div><span>{label}</span><strong>{value}</strong><small>{detail}</small></div><i><Icon size={22} /></i></article>
}

export function PageIntro({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-intro"><div>{eyebrow && <span>{eyebrow}</span>}<h2>{title}</h2><p>{description}</p></div>{action}</div>
}

export function StatusBadge({ children, tone = 'neutral' }: { children: ReactNode; tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info' }) {
  return <span className={`status-badge ${tone}`}>{children}</span>
}

export function LoadingState({ label = 'Loading secure records…' }: { label?: string }) {
  return <div className="state-box"><LoaderCircle className="spin" size={26} /><span>{label}</span></div>
}

export function EmptyState({ title = 'No records found', detail = 'Try adjusting the search criteria.' }: { title?: string; detail?: string }) {
  return <div className="state-box"><SearchX size={27} /><strong>{title}</strong><span>{detail}</span></div>
}

export function ErrorState({ message, retry }: { message: string; retry?: () => void }) {
  return <div className="state-box error"><AlertCircle size={27} /><strong>Could not load this information</strong><span>{message}</span>{retry && <button className="button secondary small" onClick={retry}>Try again</button>}</div>
}

export function Toast({ message, type = 'success' }: { message: string; type?: 'success' | 'error' }) {
  return <div className={`toast ${type}`}>{type === 'success' ? <CheckCircle2 size={18} /> : <AlertCircle size={18} />} {message}</div>
}

export function formatDate(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('en-IN', { day: '2-digit', month: 'short', year: 'numeric' }).format(date)
}

export function crimeLabel(id?: number) {
  return ({ 1: 'Property crime', 2: 'Crime against person', 3: 'Cybercrime', 4: 'Public order', 5: 'Economic offence' } as Record<number, string>)[id || 0] || `Crime category ${id || '—'}`
}
