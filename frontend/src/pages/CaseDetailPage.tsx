import { useCallback, useEffect, useState } from 'react'
import { Activity, ArrowLeft, CalendarDays, Check, Clock3, Fingerprint, MapPin, RefreshCw, ShieldAlert, UserRound } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { get, patch, post } from '../lib/api'
import type { CaseRecord, Readiness, SimilarCase } from '../types'
import { EmptyState, ErrorState, LoadingState, StatusBadge, Toast, crimeLabel, formatDate } from '../components/UI'
import { ConnectionsPanel, EvidenceWorkspace, LegalCustodyPanel, PartiesPanel, TasksPanel } from '../components/CaseWorkspacePanels'

type RecordRow = Record<string, unknown>
const pick = (row: RecordRow, ...keys: string[]) => keys.map((key) => row[key]).find((value) => value !== undefined && value !== null)

export function CaseDetailPage() {
  const { id = '' } = useParams()
  const [caseData, setCaseData] = useState<CaseRecord | null>(null)
  const [readiness, setReadiness] = useState<Readiness | null>(null)
  const [similar, setSimilar] = useState<SimilarCase[]>([])
  const [documents, setDocuments] = useState<RecordRow[]>([])
  const [timeline, setTimeline] = useState<RecordRow[]>([])
  const [activeTab, setActiveTab] = useState('overview')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [toast, setToast] = useState('')
  const load = useCallback(async () => {
    setLoading(true); setError('')
    try {
      const [record, score, related, docs, events] = await Promise.all([
        get<CaseRecord>(`/cases/${id}`), get<Readiness>(`/analytics/cases/${id}/readiness`),
        get<{ cases: SimilarCase[] }>(`/analytics/cases/${id}/similar?limit=6`), get<RecordRow[]>(`/cases/${id}/documents`), get<RecordRow[]>(`/cases/${id}/timeline`),
      ])
      setCaseData(record); setReadiness(score); setSimilar(related.cases || []); setDocuments(docs || []); setTimeline(events || [])
    } catch (err) { setError(err instanceof Error ? err.message : 'Case unavailable') } finally { setLoading(false) }
  }, [id])
  useEffect(() => { void load() }, [load])
  async function updateStatus(value: string) { if (!value) return; try { await patch(`/cases/${id}/status`, { case_status_id: Number(value) }); setToast('Case status updated'); await load() } catch (err) { setToast(err instanceof Error ? err.message : 'Status update failed') } }
  async function indexCase() { try { await post(`/search/cases/${id}/index`); setToast('Case indexed for hybrid intelligence search') } catch (err) { setToast(err instanceof Error ? err.message : 'Indexing is unavailable for this role or environment') } }
  if (loading) return <LoadingState label="Opening secured case workspace…" />
  if (error || !caseData) return <ErrorState message={error || 'Case not found'} retry={load} />
  const tabs = [['overview', 'Overview'], ['people', 'People'], ['legal', 'Legal & custody'], ['tasks', 'Tasks'], ['timeline', 'Timeline'], ['evidence', 'Evidence'], ['connections', 'Connections']]
  return <div className="page-stack case-detail-page">
    {toast && <div onClick={() => setToast('')}><Toast message={toast} type={toast.includes('failed') || toast.includes('unavailable') ? 'error' : 'success'} /></div>}
    <div className="case-title-row"><div><Link to="/cases" className="back-link"><ArrowLeft size={15} /> Back to case registry</Link><div className="case-title"><span className="case-shield"><Fingerprint size={23} /></span><div><span>CRIME NUMBER</span><h2>{caseData.CrimeNo || `Case #${id}`}</h2></div><StatusBadge tone={caseData.GravityOffenceID === 1 ? 'danger' : 'info'}>{caseData.GravityOffenceID === 1 ? 'Grave offence' : 'Under investigation'}</StatusBadge></div></div><div className="case-actions"><button className="button secondary" onClick={indexCase}><RefreshCw size={16} /> Index case</button><select className="status-select" defaultValue="" onChange={(e) => updateStatus(e.target.value)}><option value="" disabled>Update status</option><option value="1">Registered</option><option value="2">Under investigation</option><option value="3">Chargesheet filed</option><option value="4">Closed</option></select></div></div>
    <section className="case-meta glass-panel"><Meta icon={CalendarDays} label="Registered" value={formatDate(caseData.CrimeRegisteredDate)} /><Meta icon={ShieldAlert} label="Classification" value={crimeLabel(caseData.CrimeMajorHeadID)} /><Meta icon={MapPin} label="Location" value={caseData.Latitude && caseData.Longitude ? `${caseData.Latitude.toFixed(4)}, ${caseData.Longitude.toFixed(4)}` : 'Not recorded'} /><Meta icon={UserRound} label="Police unit" value={`Unit ${caseData.PoliceStationID}`} /></section>
    <nav className="tab-nav">{tabs.map(([key, label]) => <button key={key} className={activeTab === key ? 'active' : ''} onClick={() => setActiveTab(key)}>{label}{key === 'evidence' && <span>{documents.length}</span>}</button>)}</nav>
    {activeTab === 'overview' && <div className="case-content-grid"><section className="panel glass-panel"><div className="panel-head"><div><span className="panel-kicker">FIR NARRATIVE</span><h3>Brief facts</h3></div></div><p className="narrative">{caseData.BriefFacts}</p><div className="incident-grid"><div><span>Incident from</span><strong>{formatDate(caseData.IncidentFromDate)}</strong></div><div><span>Incident to</span><strong>{formatDate(caseData.IncidentToDate)}</strong></div><div><span>Major / minor head</span><strong>{caseData.CrimeMajorHeadID} / {caseData.CrimeMinorHeadID}</strong></div><div><span>Case category</span><strong>{caseData.CaseCategoryID}</strong></div></div></section><ReadinessPanel readiness={readiness} /></div>}
    {activeTab === 'people' && <PartiesPanel caseID={id} />}
    {activeTab === 'legal' && <LegalCustodyPanel caseID={id} />}
    {activeTab === 'tasks' && <TasksPanel caseID={id} />}
    {activeTab === 'timeline' && <section className="panel glass-panel"><div className="panel-head"><div><span className="panel-kicker">AUDITABLE CHRONOLOGY</span><h3>Case timeline</h3></div></div>{timeline.length === 0 ? <EmptyState title="No timeline events" /> : <div className="timeline">{timeline.map((event, index) => <div key={index}><span><Activity size={15} /></span><div><strong>{String(pick(event, 'event_type', 'EventType', 'type', 'Type') || 'Case activity')}</strong><p>{String(pick(event, 'description', 'Description', 'details', 'Details') || 'Record updated')}</p><small>{formatDate(String(pick(event, 'occurred_at', 'OccurredAt', 'date', 'Date', 'created_at', 'CreatedAt') || ''))}</small></div></div>)}</div>}</section>}
    {activeTab === 'evidence' && <EvidenceWorkspace caseID={id} />}
    {activeTab === 'connections' && <ConnectionsPanel caseID={id} similar={similar} />}
  </div>
}

function Meta({ icon: Icon, label, value }: { icon: typeof MapPin; label: string; value: string }) { return <div><i><Icon size={18} /></i><span>{label}<strong>{value}</strong></span></div> }
function ReadinessPanel({ readiness }: { readiness: Readiness | null }) { if (!readiness) return null; return <aside className="panel glass-panel readiness-panel"><div className="panel-head"><div><span className="panel-kicker">DECISION SUPPORT</span><h3>Case readiness</h3></div><StatusBadge tone={readiness.score >= 85 ? 'success' : readiness.score >= 65 ? 'warning' : 'danger'}>{readiness.band.replaceAll('-', ' ')}</StatusBadge></div><div className="readiness-score"><div className="score-ring" style={{ '--score': `${readiness.score * 3.6}deg` } as React.CSSProperties}><span><strong>{readiness.score}</strong>/100</span></div><div><strong>{readiness.checks.filter((c) => c.passed).length} of {readiness.checks.length} checks passed</strong><span>Supervisor review readiness</span></div></div><div className="check-list">{readiness.checks.map((check) => <div key={check.name} className={check.passed ? 'passed' : ''}>{check.passed ? <Check size={15} /> : <Clock3 size={15} />}<div><strong>{check.name.replaceAll('_', ' ')}</strong>{check.action && <span>{check.action}</span>}</div><small>{check.weight} pts</small></div>)}</div><p className="advisory">{readiness.disclaimer}</p></aside> }
