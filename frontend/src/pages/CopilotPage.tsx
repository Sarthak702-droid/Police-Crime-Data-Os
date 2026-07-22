import { useEffect, useRef, useState, type ChangeEvent, type FormEvent } from 'react'
import { Bot, ChevronRight, Download, FileSearch, History, Languages, MapPinned, Mic, Paperclip, Plus, Send, ShieldCheck, Sparkles, Square, UserRound } from 'lucide-react'
import { get, post, upload, API_BASE } from '../lib/api'
import type { ChatResponse, ChatTurn } from '../types'
import { EmptyState, Toast } from '../components/UI'

interface Message { role: 'user' | 'assistant'; content: string; confidence?: number; trail?: string }
interface Session { SessionID?: string; session_id?: string; CreatedAt?: string; created_at?: string }

const suggestions = [
  { icon: MapPinned, text: 'Show burglary hotspots in my jurisdiction' },
  { icon: FileSearch, text: 'Which pending cases need urgent action?' },
  { icon: ShieldCheck, text: 'Find repeat offenders with three or more cases' },
]

export function CopilotPage() {
  const [messages, setMessages] = useState<Message[]>([])
  const [sessions, setSessions] = useState<Session[]>([])
  const [sessionID, setSessionID] = useState<string>(() => crypto.randomUUID())
  const [message, setMessage] = useState('')
  const [language, setLanguage] = useState('en-IN')
  const [busy, setBusy] = useState(false)
  const [recording, setRecording] = useState(false)
  const [notice, setNotice] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const mediaRecorder = useRef<MediaRecorder | null>(null)
  const chunks = useRef<Blob[]>([])
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => { get<Session[]>('/chat/sessions').then((rows) => setSessions(rows || [])).catch(() => undefined) }, [])
  useEffect(() => endRef.current?.scrollIntoView({ behavior: 'smooth' }), [messages, busy])
  async function send(text = message) {
    const clean = text.trim(); if (!clean || busy) return
    setMessage(''); setMessages((current) => [...current, { role: 'user', content: clean }]); setBusy(true)
    try {
      const result = await post<ChatResponse>('/chat/query', { session_id: sessionID, message: clean, language })
      setMessages((current) => [...current, { role: 'assistant', content: result.answer || result.response || 'No answer returned.', confidence: result.confidence, trail: (result as unknown as Record<string, string>).evidence_trail_id }])
    } catch (err) { setMessages((current) => [...current, { role: 'assistant', content: err instanceof Error ? err.message : 'The copilot is temporarily unavailable.' }]) } finally { setBusy(false) }
  }
  async function selectSession(session: Session) {
    const id = session.SessionID || session.session_id; if (!id) return
    setSessionID(id)
    try { const turns = await get<ChatTurn[]>(`/chat/sessions/${id}/turns`); setMessages((turns || []).map((turn) => ({ role: (turn.Speaker || turn.speaker) === 'user' ? 'user' : 'assistant', content: turn.Content || turn.content || '' }))) } catch { setMessages([]) }
  }
  async function toggleRecording() {
    if (recording) { mediaRecorder.current?.stop(); setRecording(false); return }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true }); const recorder = new MediaRecorder(stream); chunks.current = []
      recorder.ondataavailable = (event) => chunks.current.push(event.data)
      recorder.onstop = async () => { stream.getTracks().forEach((track) => track.stop()); const blob = new Blob(chunks.current, { type: recorder.mimeType }); const form = new FormData(); form.append('file', blob, 'officer-query.webm'); form.append('language_code', language); form.append('mode', 'transcribe'); try { const result = await upload<{ transcript: string }>('/ai/speech-to-text', form); setMessage(result.transcript); setNotice('Voice transcribed—review before sending') } catch (err) { setNotice(err instanceof Error ? err.message : 'Transcription unavailable') } }
      mediaRecorder.current = recorder; recorder.start(); setRecording(true)
    } catch { setNotice('Microphone permission is required for voice queries') }
  }
  async function transcribeFile(event: ChangeEvent<HTMLInputElement>) { const file = event.target.files?.[0]; if (!file) return; const form = new FormData(); form.append('file', file); form.append('language_code', language); form.append('mode', 'transcribe'); try { const result = await upload<{ transcript: string }>('/ai/speech-to-text', form); setMessage(result.transcript); setNotice('Audio transcribed—review before sending') } catch (err) { setNotice(err instanceof Error ? err.message : 'Transcription unavailable') } event.target.value = '' }
  async function exportPDF() { const token = localStorage.getItem('drishti_access_token'); const response = await fetch(`${API_BASE}/chat/sessions/${sessionID}/export/pdf`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } }); if (!response.ok) { setNotice('Export failed'); return } const blob = await response.blob(); const url = URL.createObjectURL(blob); const anchor = document.createElement('a'); anchor.href = url; anchor.download = `conversation-${sessionID}.pdf`; anchor.click(); URL.revokeObjectURL(url) }
  function newChat() { setSessionID(crypto.randomUUID()); setMessages([]); setMessage('') }

  return <div className="copilot-page">
    {notice && <div onClick={() => setNotice('')}><Toast message={notice} type={notice.includes('failed') || notice.includes('required') || notice.includes('unavailable') ? 'error' : 'success'} /></div>}
    <aside className="chat-history glass-panel"><button className="button primary" onClick={newChat}><Plus size={17} /> New investigation</button><div className="history-title"><History size={15} /> RECENT SESSIONS</div>{sessions.length === 0 ? <EmptyState title="No previous sessions" detail="Your grounded queries will appear here." /> : <div className="session-list">{sessions.slice(0, 12).map((session, index) => { const id = session.SessionID || session.session_id || ''; return <button key={id || index} className={id === sessionID ? 'active' : ''} onClick={() => selectSession(session)}><Bot size={16} /><span><strong>Investigation session</strong><small>{id.slice(0, 10)}…</small></span><ChevronRight size={14} /></button> })}</div>}<div className="scope-card"><ShieldCheck size={18} /><div><strong>Jurisdiction scoped</strong><span>Copilot can only access records your role permits.</span></div></div></aside>
    <section className="chat-workspace glass-panel">
      <header className="chat-head"><div className="copilot-identity"><span><Bot size={22} /><i /></span><div><strong>Drishti Copilot</strong><small><i /> Grounded tools online</small></div></div><div><label><Languages size={16} /><select value={language} onChange={(e) => setLanguage(e.target.value)}><option value="en-IN">English</option><option value="kn-IN">ಕನ್ನಡ</option></select></label><button className="icon-button" title="Export audited PDF" onClick={exportPDF}><Download size={18} /></button></div></header>
      <div className="chat-messages">
        {messages.length === 0 && <div className="chat-welcome"><div className="ai-hero-icon"><Bot size={34} /><Sparkles size={15} /></div><span>INTELLIGENCE, GROUNDED IN YOUR RECORDS</span><h2>How can I assist your investigation?</h2><p>Ask naturally in English or Kannada. Every answer uses allowlisted tools, respects your jurisdiction and preserves an evidence trail.</p><div className="suggestion-grid">{suggestions.map(({ icon: Icon, text }) => <button key={text} onClick={() => send(text)}><Icon size={19} /><span>{text}</span><ChevronRight size={15} /></button>)}</div></div>}
        {messages.map((item, index) => <div className={`message ${item.role}`} key={index}><span className="message-avatar">{item.role === 'assistant' ? <Bot size={18} /> : <UserRound size={18} />}</span><div><small>{item.role === 'assistant' ? 'DRISHTI COPILOT' : 'YOU'}</small><p>{item.content}</p>{item.role === 'assistant' && item.confidence !== undefined && <div className="answer-meta"><ShieldCheck size={14} /> Confidence {Math.round(item.confidence * 100)}% {item.trail && <span>· Evidence {item.trail.slice(0, 8)}</span>}</div>}</div></div>)}
        {busy && <div className="message assistant"><span className="message-avatar"><Bot size={18} /></span><div><small>DRISHTI COPILOT</small><div className="typing"><i /><i /><i /></div><span className="tool-status">Checking authorized records…</span></div></div>}
        <div ref={endRef} />
      </div>
      <form className="chat-composer" onSubmit={(e: FormEvent) => { e.preventDefault(); void send() }}><textarea rows={2} value={message} onChange={(e) => setMessage(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); void send() } }} placeholder={language === 'kn-IN' ? 'ನಿಮ್ಮ ತನಿಖೆಯ ಪ್ರಶ್ನೆಯನ್ನು ಕೇಳಿ…' : 'Ask about cases, patterns, pending actions or criminal networks…'} /><div><span><input ref={fileRef} type="file" accept="audio/*" hidden onChange={transcribeFile} /><button type="button" title="Transcribe audio file" onClick={() => fileRef.current?.click()}><Paperclip size={18} /></button><button type="button" className={recording ? 'recording' : ''} title="Voice query" onClick={toggleRecording}>{recording ? <Square size={17} /> : <Mic size={18} />}</button><small>{recording ? 'Recording… tap to stop' : 'Shift + Enter for new line'}</small></span><button className="send-button" disabled={!message.trim() || busy}><Send size={18} /></button></div></form>
      <footer className="chat-disclaimer"><ShieldCheck size={13} /> Decision support only. Verify critical information against the source FIR and departmental procedure.</footer>
    </section>
  </div>
}
