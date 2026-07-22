import { useState, type FormEvent } from 'react'
import { ArrowLeft, ArrowRight, Check, FileCheck2, MapPin, Save, ShieldCheck, UserRound, UsersRound } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { post } from '../lib/api'
import type { CaseRecord } from '../types'
import { PageIntro, Toast } from '../components/UI'

const steps = [
  { label: 'Incident', icon: MapPin }, { label: 'Classification', icon: ShieldCheck }, { label: 'People', icon: UsersRound }, { label: 'Review', icon: FileCheck2 },
]

export function NewFirPage() {
  const [step, setStep] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const [form, setForm] = useState({
    incident_from_date: '', incident_to_date: '', info_received_ps_date: '', latitude: '', longitude: '', brief_facts: '',
    case_category_id: '1', gravity_offence_id: '2', crime_major_head_id: '1', crime_minor_head_id: '1', court_id: '1',
    complainant_name: '', complainant_mobile: '', victim_name: '', accused_name: '',
  })
  const update = (key: string, value: string) => setForm((current) => ({ ...current, [key]: value }))
  const formatBackendDate = (value: string) => value ? `${value.replace('T', ' ')}:00` : ''
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (step < 3) { setStep(step + 1); return }
    setBusy(true); setError('')
    try {
      const body = {
        case_category_id: Number(form.case_category_id), gravity_offence_id: Number(form.gravity_offence_id), crime_major_head_id: Number(form.crime_major_head_id), crime_minor_head_id: Number(form.crime_minor_head_id), court_id: Number(form.court_id),
        incident_from_date: formatBackendDate(form.incident_from_date), incident_to_date: formatBackendDate(form.incident_to_date), info_received_ps_date: formatBackendDate(form.info_received_ps_date),
        latitude: Number(form.latitude || 0), longitude: Number(form.longitude || 0), brief_facts: form.brief_facts,
        complainants: form.complainant_name ? [{ ComplainantName: form.complainant_name, MobileNo: form.complainant_mobile }] : [],
        victims: form.victim_name ? [{ VictimName: form.victim_name }] : [],
        accused_list: form.accused_name ? [{ AccusedName: form.accused_name }] : [],
      }
      const created = await post<CaseRecord>('/cases', body)
      navigate(`/cases/${created.CaseMasterID}`)
    } catch (err) { setError(err instanceof Error ? err.message : 'Unable to register FIR') } finally { setBusy(false) }
  }

  return <div className="page-stack new-fir-page">
    <PageIntro eyebrow="STRUCTURED REGISTRATION" title="Register new FIR" description="Required information is validated before the record is submitted." action={<Link className="button secondary" to="/cases"><ArrowLeft size={17} /> Exit registration</Link>} />
    {error && <Toast message={error} type="error" />}
    <div className="fir-layout">
      <aside className="fir-progress glass-panel">{steps.map(({ label, icon: Icon }, index) => <button key={label} type="button" onClick={() => index < step && setStep(index)} className={`${step === index ? 'active' : ''} ${index < step ? 'done' : ''}`}><span>{index < step ? <Check size={16} /> : <Icon size={17} />}</span><div><small>STEP {index + 1}</small><strong>{label}</strong></div></button>)}<div className="draft-note"><Save size={17} /><div><strong>Secure submission</strong><span>Record will be scoped to your current unit.</span></div></div></aside>
      <form className="fir-form glass-panel" onSubmit={submit}>
        {step === 0 && <fieldset><legend>Incident information</legend><p>Record when, where and how the reported incident occurred.</p><div className="form-grid"><label>Incident from *<input type="datetime-local" required value={form.incident_from_date} onChange={(e) => update('incident_from_date', e.target.value)} /></label><label>Incident to *<input type="datetime-local" required value={form.incident_to_date} onChange={(e) => update('incident_to_date', e.target.value)} /></label><label>Information received at station *<input type="datetime-local" required value={form.info_received_ps_date} onChange={(e) => update('info_received_ps_date', e.target.value)} /></label><div /><label>Latitude<input type="number" step="any" value={form.latitude} onChange={(e) => update('latitude', e.target.value)} placeholder="12.9716" /></label><label>Longitude<input type="number" step="any" value={form.longitude} onChange={(e) => update('longitude', e.target.value)} placeholder="77.5946" /></label><label className="full">Brief facts *<textarea required minLength={20} rows={7} value={form.brief_facts} onChange={(e) => update('brief_facts', e.target.value)} placeholder="Describe what happened, when, where, how, and any reported loss or injury…" /><small>{form.brief_facts.length} characters</small></label></div></fieldset>}
        {step === 1 && <fieldset><legend>Case classification</legend><p>Apply the appropriate case, gravity, crime-head and court references.</p><div className="form-grid"><label>Case category ID *<input type="number" min="1" required value={form.case_category_id} onChange={(e) => update('case_category_id', e.target.value)} /></label><label>Gravity *<select value={form.gravity_offence_id} onChange={(e) => update('gravity_offence_id', e.target.value)}><option value="1">Grave offence</option><option value="2">Non-grave offence</option></select></label><label>Major crime head ID *<input type="number" min="1" required value={form.crime_major_head_id} onChange={(e) => update('crime_major_head_id', e.target.value)} /></label><label>Minor crime head ID *<input type="number" min="1" required value={form.crime_minor_head_id} onChange={(e) => update('crime_minor_head_id', e.target.value)} /></label><label>Court ID *<input type="number" min="1" required value={form.court_id} onChange={(e) => update('court_id', e.target.value)} /></label></div><div className="info-callout"><ShieldCheck size={18} /><div><strong>Legal sections can be added in the case workspace</strong><span>After FIR registration, attach applicable acts and sections with their current legal references.</span></div></div></fieldset>}
        {step === 2 && <fieldset><legend>People involved</legend><p>Add the known complainant, victim and accused details. More records can be added later.</p><div className="people-section"><h4><UserRound size={17} /> Complainant</h4><div className="form-grid"><label>Full name<input value={form.complainant_name} onChange={(e) => update('complainant_name', e.target.value)} placeholder="Complainant name" /></label><label>Mobile number<input value={form.complainant_mobile} onChange={(e) => update('complainant_mobile', e.target.value)} placeholder="10-digit mobile number" /></label></div></div><div className="people-section"><h4><UserRound size={17} /> Victim and accused</h4><div className="form-grid"><label>Victim name<input value={form.victim_name} onChange={(e) => update('victim_name', e.target.value)} placeholder="Victim name, if applicable" /></label><label>Known accused<input value={form.accused_name} onChange={(e) => update('accused_name', e.target.value)} placeholder="Name or Unknown Person" /></label></div></div></fieldset>}
        {step === 3 && <fieldset><legend>Review and register</legend><p>Confirm the information below before creating the jurisdictional record.</p><div className="review-grid"><Review label="Incident period" value={`${form.incident_from_date || '—'} to ${form.incident_to_date || '—'}`} /><Review label="Information received" value={form.info_received_ps_date || '—'} /><Review label="Classification" value={`Category ${form.case_category_id} · Major ${form.crime_major_head_id} · Minor ${form.crime_minor_head_id}`} /><Review label="Gravity" value={form.gravity_offence_id === '1' ? 'Grave offence' : 'Non-grave offence'} /><Review label="Location" value={form.latitude && form.longitude ? `${form.latitude}, ${form.longitude}` : 'Not provided'} /><Review label="People" value={[form.complainant_name, form.victim_name, form.accused_name].filter(Boolean).join(' · ') || 'To be added later'} /><Review label="Brief facts" value={form.brief_facts || '—'} full /></div><label className="certify"><input type="checkbox" required /> I confirm this information reflects the report received and understand that the submission is audited.</label></fieldset>}
        <div className="form-actions"><button type="button" className="button secondary" disabled={step === 0} onClick={() => setStep(step - 1)}><ArrowLeft size={17} /> Back</button><button className="button primary" disabled={busy}>{busy ? 'Registering…' : step === 3 ? 'Register FIR' : 'Continue'}{step === 3 ? <FileCheck2 size={17} /> : <ArrowRight size={17} />}</button></div>
      </form>
    </div>
  </div>
}

function Review({ label, value, full = false }: { label: string; value: string; full?: boolean }) { return <div className={full ? 'full' : ''}><span>{label}</span><strong>{value}</strong></div> }
