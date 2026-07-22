import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { BrainCircuit, CalendarDays, ChevronLeft, ChevronRight, FilePlus2, FolderSearch2, Search, SlidersHorizontal, X } from 'lucide-react'
import { Link, useSearchParams } from 'react-router-dom'
import { get } from '../lib/api'
import type { CaseRecord, SearchResult } from '../types'
import { EmptyState, ErrorState, LoadingState, PageIntro, StatusBadge, crimeLabel, formatDate } from '../components/UI'

export function CasesPage() {
  const [params, setParams] = useSearchParams()
  const [input, setInput] = useState(params.get('keyword') || '')
  const [cases, setCases] = useState<CaseRecord[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const page = Number(params.get('page') || '1')
  const hybrid = params.get('mode') === 'hybrid'
  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const query = new URLSearchParams(params); query.set('limit', '12')
      if (hybrid && params.get('keyword')) {
        const hits = await get<Array<{ case_master_id: number; crime_no: string; brief_facts: string; score: number }>>(`/search/hybrid?q=${encodeURIComponent(params.get('keyword') || '')}&limit=50`)
        setCases(hits.map((hit) => ({ CaseMasterID: hit.case_master_id, CrimeNo: hit.crime_no, BriefFacts: hit.brief_facts, CrimeRegisteredDate: '', CaseCategoryID: 0, GravityOffenceID: 0, CrimeMajorHeadID: 0, CrimeMinorHeadID: 0, PoliceStationID: 0, Latitude: 0, Longitude: 0, semanticScore: hit.score } as CaseRecord))); setTotal(hits.length)
      } else {
        const data = await get<SearchResult>(`/cases/search?${query}`)
        setCases(data.cases || []); setTotal(data.total || 0)
      }
    } catch (err) { setError(err instanceof Error ? err.message : 'Search unavailable') } finally { setLoading(false) }
  }, [params, hybrid])
  useEffect(() => { void load() }, [load])
  function submit(event: FormEvent) { event.preventDefault(); const next = new URLSearchParams(params); if (input) next.set('keyword', input); else next.delete('keyword'); next.set('page', '1'); setParams(next) }
  function setFilter(key: string, value: string) { const next = new URLSearchParams(params); if (value) next.set(key, value); else next.delete(key); next.set('page', '1'); setParams(next) }
  const pages = Math.max(1, Math.ceil(total / 12))

  return <div className="page-stack cases-page">
    <PageIntro eyebrow="FIR & INVESTIGATION RECORDS" title="Case registry" description="Search jurisdiction-authorized case records and continue investigation work." action={<Link className="button primary" to="/cases/new"><FilePlus2 size={17} /> Register FIR</Link>} />
    <section className="search-toolbar glass-panel">
      <form onSubmit={submit}><Search size={19} /><input id="global-search" value={input} onChange={(e) => setInput(e.target.value)} placeholder="Crime number, keywords or narrative…" /><button className="button primary">Search</button></form>
      <button className={`button secondary ${filtersOpen ? 'active' : ''}`} onClick={() => setFiltersOpen(!filtersOpen)}><SlidersHorizontal size={17} /> Filters</button>
    </section>
    <div className="search-mode"><button className={!hybrid ? 'active' : ''} onClick={() => setFilter('mode', '')}><Search size={14} /> Registry search</button><button className={hybrid ? 'active' : ''} onClick={() => setFilter('mode', 'hybrid')}><BrainCircuit size={14} /> Hybrid semantic search <span>AI</span></button><small>{hybrid ? 'Finds meaning and modus-operandi similarity, not just exact words.' : 'Exact fields, structured filters and narrative keywords.'}</small></div>
    {filtersOpen && <section className="filter-bar glass-panel"><div><label>Case status<select value={params.get('status_id') || ''} onChange={(e) => setFilter('status_id', e.target.value)}><option value="">All statuses</option><option value="1">Registered</option><option value="2">Under investigation</option><option value="3">Chargesheet filed</option><option value="4">Closed</option></select></label><label>Gravity<select value={params.get('gravity_id') || ''} onChange={(e) => setFilter('gravity_id', e.target.value)}><option value="">All gravity levels</option><option value="1">Grave</option><option value="2">Non-grave</option></select></label><label>From date<input type="date" value={params.get('from_date') || ''} onChange={(e) => setFilter('from_date', e.target.value)} /></label><label>To date<input type="date" value={params.get('to_date') || ''} onChange={(e) => setFilter('to_date', e.target.value)} /></label></div><button className="text-button" onClick={() => { setParams({}); setInput('') }}><X size={15} /> Clear all</button></section>}
    <section className="registry-panel glass-panel">
      <div className="registry-summary"><div><FolderSearch2 size={18} /><strong>{total}</strong> records found</div><span>{hybrid ? 'BM25 + multilingual vector relevance · jurisdiction scoped' : 'Scoped to your assigned jurisdiction'}</span></div>
      {loading ? <LoadingState /> : error ? <ErrorState message={error} retry={load} /> : cases.length === 0 ? <EmptyState title="No matching cases" detail="Try a broader keyword or remove one of the filters." /> : <div className="registry-table-wrap"><table className="registry-table"><thead><tr><th>Crime number</th><th>Registered</th><th>Classification</th><th>Brief facts</th><th>Gravity</th><th>Status</th><th /></tr></thead><tbody>{cases.map((item) => <tr key={item.CaseMasterID}><td><Link to={`/cases/${item.CaseMasterID}`}><strong>{item.CrimeNo || `Case #${item.CaseMasterID}`}</strong><small>ID {item.CaseMasterID}</small></Link></td><td><CalendarDays size={14} />{formatDate(item.CrimeRegisteredDate)}</td><td>{crimeLabel(item.CrimeMajorHeadID)}<small>Head {item.CrimeMinorHeadID}</small></td><td className="facts-cell">{item.BriefFacts || 'Not recorded'}</td><td><StatusBadge tone={item.GravityOffenceID === 1 ? 'danger' : 'neutral'}>{item.GravityOffenceID === 1 ? 'Grave' : 'Non-grave'}</StatusBadge></td><td><StatusBadge tone="info">Under investigation</StatusBadge></td><td><Link className="table-open" to={`/cases/${item.CaseMasterID}`}><ChevronRight size={17} /></Link></td></tr>)}</tbody></table></div>}
      <div className="pagination"><span>Page {page} of {pages}</span><div><button disabled={page <= 1} onClick={() => setFilter('page', String(page - 1))}><ChevronLeft size={16} /> Previous</button><button disabled={page >= pages} onClick={() => setFilter('page', String(page + 1))}>Next <ChevronRight size={16} /></button></div></div>
    </section>
  </div>
}
