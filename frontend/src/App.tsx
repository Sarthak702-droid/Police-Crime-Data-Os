import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './state/AuthContext'
import { AppShell } from './components/AppShell'
import { LoginPage } from './pages/LoginPage'
import { DashboardPage } from './pages/DashboardPage'
import { CasesPage } from './pages/CasesPage'
import { CaseDetailPage } from './pages/CaseDetailPage'
import { NewFirPage } from './pages/NewFirPage'
import { CopilotPage } from './pages/CopilotPage'
import { IntelligencePage } from './pages/IntelligencePage'

function ProtectedApp() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<DashboardPage />} />
        <Route path="/cases" element={<CasesPage />} />
        <Route path="/cases/new" element={<NewFirPage />} />
        <Route path="/cases/:id" element={<CaseDetailPage />} />
        <Route path="/copilot" element={<CopilotPage />} />
        <Route path="/intelligence" element={<IntelligencePage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppShell>
  )
}

export default function App() {
  const { officer, loading } = useAuth()
  if (loading) return <div className="boot-screen"><div className="brand-mark">ದೃ</div><span>Securing your workspace…</span></div>
  if (!officer) return <Routes><Route path="*" element={<LoginPage />} /></Routes>
  return <ProtectedApp />
}
