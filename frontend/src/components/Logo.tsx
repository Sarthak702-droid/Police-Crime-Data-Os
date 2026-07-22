import { ShieldCheck } from 'lucide-react'

export function Logo({ compact = false }: { compact?: boolean }) {
  return (
    <div className="logo-wrap">
      <div className="logo-emblem"><ShieldCheck size={compact ? 21 : 25} strokeWidth={1.8} /></div>
      {!compact && <div><strong>DRISHTI</strong><span>POLICE INTELLIGENCE</span></div>}
    </div>
  )
}
