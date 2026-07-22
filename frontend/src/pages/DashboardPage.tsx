import { useCallback, useEffect, useMemo, useState } from 'react'
import { ArrowRight, Bot, CheckCircle2, Clock3, FilePlus2, FolderOpen, MapPinned, Search, ShieldAlert, Sparkles, TriangleAlert } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { get } from '../lib/api'
import type { CaseRecord, PendingCase, SearchResult } from '../types'
import { EmptyState, ErrorState, LoadingState, PageIntro, StatCard, StatusBadge, crimeLabel, formatDate } from '../components/UI'

interface Hotspot { area?: string; Area?: string; count?: number; Count?: number; latitude?: number; longitude?: number }

export function DashboardPage() {
  const [cases, setCases] = useState<CaseRecord[]>([])
  const [total, setTotal] = useState(0)
  const [pending, setPending] = useState<PendingCase[]>([])
  const [hotspots, setHotspots] = useState<Hotspot[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const navigate = useNavigate()
  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const [caseData, pendingData, hotspotData] = await Promise.all([
        get<SearchResult>('/cases/search?limit=6&page=1'),
        get<{ cases: PendingCase[] }>('/analytics/pending-actions?minimum_age_days=15'),
        get<Hotspot[]>('/analytics/hotspots'),
      ])
      setCases(caseData.cases || []); setTotal(caseData.total || 0); setPending(pendingData.cases || []); setHotspots(hotspotData || [])
    } catch (err) { setError(err instanceof Error ? err.message : 'Dashboard unavailable') } finally { setLoading(false) }
  }, [])
  useEffect(() => { void load() }, [load])
  const severe = useMemo(() => cases.filter((item) => item.GravityOffenceID === 1).length, [cases])
  const urgent = pending.filter((item) => item.priority_score >= 75).length

  if (loading) return <LoadingState label="Building your operational picture…" />
  if (error) return <ErrorState message={error} retry={load} />
  return <div className="dashboard-page page-stack">
    <PageIntro eyebrow="LIVE OPERATIONS" title="Good day, Officer" description="A jurisdiction-scoped view of cases that need attention now." action={<div className="intro-actions"><Link className="button secondary" to="/cases"><Search size={17} /> Search cases</Link><Link className="button primary" to="/cases/new"><FilePlus2 size={17} /> Register FIR</Link></div>} />
    <form className="command-search glass-panel" onSubmit={(e) => { e.preventDefault(); navigate(`/cases?keyword=${encodeURIComponent(query)}`) }}><Search size={20} /><input id="global-search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search by crime number, facts, location or keyword…" /><button>Search registry</button></form>
    <section className="stat-grid">
      <StatCard label="TOTAL CASE RECORDS" value={total} detail="Within current jurisdiction" icon={FolderOpen} />
      <StatCard label="PENDING ACTIONS" value={pending.length} detail={`${urgent} high priority`} icon={Clock3} tone="amber" />
      <StatCard label="GRAVE CASES" value={severe} detail="In latest records" icon={ShieldAlert} tone="red" />
      <StatCard label="ACTIVE HOTSPOTS" value={hotspots.length} detail="Burglary clusters detected" icon={MapPinned} tone="blue" />
    </section>
    <section className="dashboard-grid">
      <article className="panel glass-panel recent-cases">
        <div className="panel-head"><div><span className="panel-kicker">CASE REGISTRY</span><h3>Recently registered</h3></div><Link to="/cases">View all <ArrowRight size={15} /></Link></div>
        {cases.length === 0 ? <EmptyState title="No cases registered" /> : <div className="case-list">{cases.map((item) => <Link to={`/cases/${item.CaseMasterID}`} className="case-row" key={item.CaseMasterID}><div className="case-symbol">{String(item.CrimeNo || item.CaseMasterID).slice(-2)}</div><div className="case-main"><strong>{item.CrimeNo || `Case #${item.CaseMasterID}`}</strong><span>{crimeLabel(item.CrimeMajorHeadID)} · {formatDate(item.CrimeRegisteredDate)}</span><p>{item.BriefFacts || 'No brief facts available'}</p></div><StatusBadge tone={item.GravityOffenceID === 1 ? 'danger' : 'info'}>{item.GravityOffenceID === 1 ? 'Grave' : 'Under investigation'}</StatusBadge><ArrowRight className="row-arrow" size={17} /></Link>)}</div>}
      </article>
      <article className="panel glass-panel attention-panel">
        <div className="panel-head"><div><span className="panel-kicker amber-text">ACTION CENTRE</span><h3>Needs attention</h3></div><Link to="/intelligence">Prioritise <ArrowRight size={15} /></Link></div>
        {pending.length === 0 ? <div className="all-clear"><CheckCircle2 /><strong>No overdue actions</strong><span>Investigation queue is currently clear.</span></div> : <div className="attention-list">{pending.slice(0, 5).map((item, index) => <Link to={`/cases/${item.case_master_id}`} key={item.case_master_id}><span className={`priority-num p${Math.min(index + 1, 3)}`}>{index + 1}</span><div><strong>{item.crime_no}</strong><small>{item.missing_actions[0] || 'Supervisor review due'}</small></div><span className="age"><b>{item.age_days}</b> days</span></Link>)}</div>}
      </article>
    </section>
    <section className="copilot-banner glass-panel"><div className="copilot-orb"><Bot size={27} /><Sparkles size={13} /></div><div><span>DRISHTI AI COPILOT</span><h3>Ask your records, not another dashboard.</h3><p>Investigate patterns, locate related cases or check case readiness using grounded, auditable answers.</p></div><Link className="button ai-button" to="/copilot">Open copilot <ArrowRight size={17} /></Link><TriangleAlert className="banner-watermark" /></section>
  </div>
}
