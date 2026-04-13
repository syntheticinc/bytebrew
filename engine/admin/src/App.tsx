import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthContext, useAuthProvider } from './hooks/useAuth';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import HealthPage from './pages/HealthPage';
import MCPPage from './pages/MCPPage';
import ModelsPage from './pages/ModelsPage';
import SettingsPage from './pages/SettingsPage';
import APIKeysPage from './pages/APIKeysPage';
import ConfigPage from './pages/ConfigPage';
import AuditPage from './pages/AuditPage';
import AgentBuilderPage from './pages/AgentBuilderPage';
import AgentDrillInPage from './pages/AgentDrillInPage';
import AgentsPage from './pages/AgentsPage';
import SchemaListPage from './pages/SchemaListPage';
import WidgetConfigPage from './pages/WidgetConfigPage';
import KnowledgePage from './pages/KnowledgePage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('jwt');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  const auth = useAuthProvider();

  return (
    <AuthContext.Provider value={auth}>
      <BrowserRouter basename={import.meta.env.BASE_URL}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route path="/health" element={<HealthPage />} />
            <Route path="/builder" element={<SchemaListPage />} />
            <Route path="/builder/:schemaName" element={<AgentBuilderPage />} />
            <Route path="/builder/:schema/:agent" element={<AgentDrillInPage />} />
            <Route path="/mcp" element={<MCPPage />} />
            <Route path="/models" element={<ModelsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/api-keys" element={<APIKeysPage />} />
            <Route path="/config" element={<ConfigPage />} />
            <Route path="/audit" element={<AuditPage />} />
            <Route path="/knowledge" element={<KnowledgePage />} />
            <Route path="/widget" element={<WidgetConfigPage />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/agents/:agent" element={<AgentDrillInPage />} />
            <Route path="/" element={<Navigate to="/builder" replace />} />
          </Route>
          <Route path="*" element={<Navigate to="/builder" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthContext.Provider>
  );
}
