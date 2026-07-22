import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { ArrowRight, Bot, Clock3, MapPin, Network, Search, ShieldAlert, Sparkles, Target, TriangleAlert, UsersRound } from 'lucide-react'
import { Link } from 'react-router-dom'
import { get, patch } from '../lib/api'
import type { PendingCase } from '../types'
import { EmptyState, ErrorState, LoadingState, PageIntro, StatCard } from '../components/UI'

interface Hotspot { Week?: string; week?: string; CaseCount?: number; case_count?: number; PoliceStationID?: number; police_station_id?: number }
interface Node { id?: string | number; ID?: string | number; label?: string; Label?: string; type?: string; Type?: string }
interface Edge { source?: string | number; Source?: string | number; target?: string | number; Target?: string | number }
interface AssignedTask { task_id: number; case_master_id: number; title: string; priority: string; status: string; due_at: string; assignee?: { first_name?: string } }

export function IntelligencePage() {
  const [pending, setPending] = useState<PendingCase[]>([])
  const [hotspots, setHotspots] = useState<Hotspot[]>([])
  const [nodes, setNodes] = useState<Node[]>([])
  const [edges, setEdges] = useState<Edge[]>([])
  const [accusedID, setAccusedID] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [graphError, setGraphError] = useState('')
  const [tasks, setTasks] = useState<AssignedTask[]>([])
  const load = useCallback(async () => { setLoading(true); setError(''); try { const [p, h, t] = await Promise.all([get<{ cases: PendingCase[] }>('/analytics/pending-actions?minimum_age_days=30'), get<Hotspot[]>('/analytics/hotspots'), get<AssignedTask[]>('/investigation/tasks')]); setPending(p.cases || []); setHotspots(h || []); setTasks(t || []) } catch (err) { setError(err instanceof Error ? err.message : 'Intelligence data unavailable') } finally { setLoading(false) } }, [])
  useEffect(() => { void load() }, [load])
  async function loadGraph(event: FormEvent) { event.preventDefault(); setGraphError(''); try { const result = await get<{ nodes: Node[]; edges: Edge[] }>(`/graph/subgraph?accused_id=${encodeURIComponent(accusedID)}`); setNodes(result.nodes || []); setEdges(result.edges || []) } catch (err) { setGraphError(err instanceof Error ? err.message : 'Graph unavailable'); setNodes([]); setEdges([]) } }
  async function advanceTask(task: AssignedTask) { await patch(`/cases/${task.case_master_id}/tasks/${task.task_id}`, { status: task.status === 'open' ? 'in_progress' : 'completed', completion_note: task.status === 'open' ? '' : 'Completed from intelligence action centre' }); void load() }
  const maxHotspot = Math.max(1, ...hotspots.map((item) => item.CaseCount || item.case_count || 0))
  const urgent = pending.filter((item) => item.priority_score >= 75).length
  const averageAge = pending.length ? Math.round(pending.reduce((sum, item) => sum + item.age_days, 0) / pending.length) : 0
  const positions = useMemo(() => nodes.map((node, index) => ({ ...node, x: 50 + 38 * Math.cos((index / Math.max(nodes.length, 1)) * Math.PI * 2), y: 50 + 38 * Math.sin((index / Math.max(nodes.length, 1)) * Math.PI * 2) })), [nodes])
  if (loading) return <LoadingState label="Computing jurisdiction intelligence…" />
  if (error) return <ErrorState message={error} retry={load} />
  return <div className="page-stack intelligence-page">
    <PageIntro eyebrow="EXPLAINABLE DECISION SUPPORT" title="Crime intelligence" description="Prioritise investigation work and identify relationships without leaving your authorized scope." action={<Link className="button ai-button" to="/copilot"><Bot size={17} /> Ask copilot</Link>} />
    <section className="stat-grid three"><StatCard label="OVERDUE CASES" value={pending.length} detail="Older than 30 days" icon={Clock3} tone="amber" /><StatCard label="HIGH PRIORITY" value={urgent} detail="Priority score 75+" icon={ShieldAlert} tone="red" /><StatCard label="AVERAGE PENDENCY" value={`${averageAge}d`} detail="Across current queue" icon={Target} tone="blue" /></section>
    <section className="intelligence-grid">
      <article className="panel glass-panel hotspot-panel"><div className="panel-head"><div><span className="panel-kicker">SPATIO-TEMPORAL SIGNAL</span><h3>Burglary hotspot activity</h3></div><span className="live-chip"><i /> LIVE DATA</span></div>{hotspots.length === 0 ? <EmptyState title="No hotspot activity" /> : <div className="hotspot-chart">{hotspots.slice(-10).map((item, index) => { const count = item.CaseCount || item.case_count || 0; return <div key={index} title={`${count} cases`}><span style={{ height: `${Math.max(8, count / maxHotspot * 100)}%` }}><i>{count}</i></span><small>{item.Week || item.week ? new Date(item.Week || item.week || '').toLocaleDateString('en-IN', { day: '2-digit', month: 'short' }) : `W${index + 1}`}</small></div> })}</div>}<div className="chart-note"><MapPin size={15} /> Counts are limited to the authenticated police unit.</div></article>
      <article className="panel glass-panel priority-panel"><div className="panel-head"><div><span className="panel-kicker amber-text">INVESTIGATION QUEUE</span><h3>Priority actions</h3></div></div>{pending.length === 0 ? <EmptyState title="No overdue actions" /> : <div className="priority-table">{pending.slice(0, 7).map((item, index) => <Link to={`/cases/${item.case_master_id}`} key={item.case_master_id}><span className={`risk-marker ${item.priority_score >= 75 ? 'high' : ''}`}>{index + 1}</span><div><strong>{item.crime_no}</strong><small>{item.missing_actions.join(' · ') || 'Supervisor review'}</small></div><span><b>{item.priority_score}</b> priority<small>{item.age_days} days old</small></span><ArrowRight size={15} /></Link>)}</div>}<div className="advisory"><TriangleAlert size={14} /> Advisory ordering only; supervisors retain decision authority.</div></article>
    </section>
    <section className="panel glass-panel assigned-queue"><div className="panel-head"><div><span className="panel-kicker">OWNERSHIP & DEADLINES</span><h3><Clock3 size={17} /> Assigned investigation work</h3></div><span className="live-chip"><i /> {tasks.filter((task) => task.status !== 'completed').length} ACTIVE</span></div>{tasks.length === 0 ? <EmptyState title="No tasks assigned" detail="Open a case and turn an advisory missing action into accountable work." /> : <div className="assigned-task-table">{tasks.map((task) => <div key={task.task_id}><StatusBadgeForTask priority={task.priority} /><div><Link to={`/cases/${task.case_master_id}`}><strong>{task.title}</strong></Link><small>Case #{task.case_master_id} · {task.assignee?.first_name || 'Unassigned officer'}</small></div><span><b>{new Date(task.due_at).toLocaleDateString('en-IN', { day: '2-digit', month: 'short' })}</b><small>{task.status.replace('_', ' ')}</small></span>{task.status !== 'completed' && <button onClick={() => advanceTask(task)}>{task.status === 'open' ? 'Start' : 'Complete'} <ArrowRight size={13} /></button>}</div>)}</div>}</section>
    <section className="panel glass-panel network-panel"><div className="panel-head"><div><span className="panel-kicker">CO-ACCUSAL RELATIONSHIPS</span><h3>Criminal network explorer</h3></div><form onSubmit={loadGraph}><Search size={16} /><input type="number" min="1" required value={accusedID} onChange={(e) => setAccusedID(e.target.value)} placeholder="Enter accused ID" /><button>Build network</button></form></div>{graphError ? <ErrorState message={graphError} /> : nodes.length === 0 ? <div className="network-empty"><div><Network size={34} /><Sparkles size={14} /></div><strong>Explore known criminal connections</strong><span>Enter an accused ID to view cases and co-accused relationships authorized for your unit.</span></div> : <div className="network-view"><svg viewBox="0 0 100 100" role="img" aria-label="Co-accusal network">{edges.map((edge, index) => { const source = positions.find((p) => String(p.id || p.ID) === String(edge.source || edge.Source)); const target = positions.find((p) => String(p.id || p.ID) === String(edge.target || edge.Target)); return source && target ? <line key={index} x1={source.x} y1={source.y} x2={target.x} y2={target.y} /> : null })}{positions.map((node, index) => <g key={String(node.id || node.ID || index)}><circle cx={node.x} cy={node.y} r={index === 0 ? 7 : 5} className={(node.type || node.Type) === 'Person' ? 'person' : 'case'} /><text x={node.x} y={node.y + 10}>{String(node.label || node.Label || node.id || node.ID).slice(0, 18)}</text></g>)}</svg><div className="network-legend"><span><i className="person" /> Person / accused</span><span><i className="case" /> Case record</span><span><UsersRound size={15} /> {nodes.length} nodes · {edges.length} links</span></div></div>}</section>
  </div>
}

function StatusBadgeForTask({ priority }: { priority: string }) { return <span className={`task-priority-dot ${priority}`}>{priority.slice(0, 1).toUpperCase()}</span> }
