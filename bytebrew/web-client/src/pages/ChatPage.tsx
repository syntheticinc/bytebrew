import { useState, useEffect, useRef, useCallback } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { useChat } from '../hooks/useChat';
import { useAuth } from '../hooks/useAuth';
import { ChatMessage } from '../components/ChatMessage';
import { ChatInput } from '../components/ChatInput';
import { AgentSelector } from '../components/AgentSelector';
import { SessionSidebar } from '../components/SessionSidebar';
import { ThinkingIndicator } from '../components/ThinkingIndicator';
import type { AgentInfo, SessionResponse } from '../types';

export function ChatPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { logout } = useAuth();

  // Agents
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [agentsLoading, setAgentsLoading] = useState(true);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(
    searchParams.get('agent') || sessionStorage.getItem('bytebrew_last_agent'),
  );

  // Sessions
  const [sessions, setSessions] = useState<SessionResponse[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [sessionsLoading, setSessionsLoading] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-title callback: update session title from first user message
  const handleFirstAssistantResponse = useCallback(
    (firstUserMessage: string) => {
      if (!activeSessionId) return;

      // Find the session — only update if still "New chat" or empty
      const session = sessions.find((s) => s.id === activeSessionId);
      if (!session || (session.title !== 'New chat' && session.title !== '')) return;

      const title = firstUserMessage.length > 50
        ? firstUserMessage.slice(0, 47) + '...'
        : firstUserMessage;

      api.updateSession(activeSessionId, { title }).then((updated) => {
        setSessions((prev) =>
          prev.map((s) => (s.id === updated.id ? updated : s)),
        );
      }).catch(() => {
        // silent — title update is best-effort
      });
    },
    [activeSessionId, sessions],
  );

  const { messages, streaming, loadingHistory, send, respond, stop, clear } = useChat({
    currentAgent: selectedAgent,
    sessionId: activeSessionId,
    onFirstAssistantResponse: handleFirstAssistantResponse,
  });

  // Load agents on mount
  useEffect(() => {
    api.listAgents().then((data) => {
      const filtered = data.filter((a) => a.name);
      setAgents(filtered);
      if (!selectedAgent && filtered.length > 0 && filtered[0]) {
        setSelectedAgent(filtered[0].name);
      }
      setAgentsLoading(false);
    }).catch(() => {
      setAgentsLoading(false);
    });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Load sessions when agent changes
  useEffect(() => {
    if (!selectedAgent) {
      setSessions([]);
      setActiveSessionId(null);
      return;
    }

    setSessionsLoading(true);
    api.listSessions(selectedAgent).then((resp) => {
      const sorted = [...resp.data].sort(
        (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
      );
      setSessions(sorted);

      // Restore last active session for this agent, or pick the first one
      const lastKey = `bytebrew_last_session_${selectedAgent}`;
      const lastId = sessionStorage.getItem(lastKey);
      const found = lastId ? sorted.find((s) => s.id === lastId) : undefined;
      if (found) {
        setActiveSessionId(found.id);
      } else if (sorted.length > 0 && sorted[0]) {
        setActiveSessionId(sorted[0].id);
      } else {
        setActiveSessionId(null);
      }
      setSessionsLoading(false);
    }).catch(() => {
      setSessions([]);
      setActiveSessionId(null);
      setSessionsLoading(false);
    });
  }, [selectedAgent]);

  // Persist active session selection
  useEffect(() => {
    if (selectedAgent && activeSessionId) {
      sessionStorage.setItem(`bytebrew_last_session_${selectedAgent}`, activeSessionId);
    }
  }, [selectedAgent, activeSessionId]);

  // Auto-scroll on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSelectAgent = useCallback(
    (name: string) => {
      setSelectedAgent(name);
      setSearchParams({ agent: name });
      sessionStorage.setItem('bytebrew_last_agent', name);
      setSidebarOpen(false);
    },
    [setSearchParams],
  );

  const handleSelectSession = useCallback((id: string) => {
    setActiveSessionId(id);
    setSidebarOpen(false);
  }, []);

  const handleNewSession = useCallback(() => {
    if (!selectedAgent) return;
    api.createSession({ title: 'New chat', agent_name: selectedAgent }).then((created) => {
      setSessions((prev) => [created, ...prev]);
      setActiveSessionId(created.id);
    }).catch(() => {
      // silent
    });
  }, [selectedAgent]);

  const handleRenameSession = useCallback((id: string, title: string) => {
    api.updateSession(id, { title }).then((updated) => {
      setSessions((prev) =>
        prev.map((s) => (s.id === updated.id ? updated : s)),
      );
    }).catch(() => {
      // silent
    });
  }, []);

  const handleDeleteSession = useCallback(
    (id: string) => {
      api.deleteSession(id).then(() => {
        setSessions((prev) => {
          const next = prev.filter((s) => s.id !== id);
          // If deleted session was active, switch to most recent
          if (id === activeSessionId) {
            const first = next[0];
            setActiveSessionId(first ? first.id : null);
          }
          return next;
        });
      }).catch(() => {
        // silent
      });
    },
    [activeSessionId],
  );

  const handleSend = useCallback(
    (text: string) => {
      if (!selectedAgent) return;

      // If no active session, create one first
      if (!activeSessionId) {
        api.createSession({ title: 'New chat', agent_name: selectedAgent }).then((created) => {
          setSessions((prev) => [created, ...prev]);
          setActiveSessionId(created.id);
          // Send will be triggered by the session change + messages effect
          // But we need to send immediately, so we call send after state updates
          // Using setTimeout to let React commit the state
          setTimeout(() => {
            send(selectedAgent, text);
          }, 0);
        }).catch(() => {
          // silent
        });
        return;
      }

      send(selectedAgent, text);
    },
    [selectedAgent, activeSessionId, send],
  );

  const handleConfirmRespond = useCallback(
    (answer: 'yes' | 'no') => {
      if (!selectedAgent) return;
      respond(selectedAgent, answer);
    },
    [selectedAgent, respond],
  );

  const handleClear = useCallback(() => {
    clear();
  }, [clear]);

  const activeSession = sessions.find((s) => s.id === activeSessionId);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex h-screen bg-brand-dark">
      {/* Mobile backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={`
        fixed inset-y-0 left-0 z-50 flex w-60 flex-col border-r border-brand-shade3/15 bg-brand-dark transition-transform duration-200
        md:relative md:translate-x-0 md:flex-shrink-0
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        {/* Logo + Logout */}
        <div className="flex items-center justify-between border-b border-brand-shade3/15 px-4 py-3">
          <h1 className="text-sm font-bold text-brand-light">
            Byte<span className="text-brand-accent">Brew</span>
          </h1>
          <button
            onClick={logout}
            className="text-xs text-brand-shade3 transition-colors hover:text-brand-light"
          >
            Logout
          </button>
        </div>

        {/* Agents section */}
        <div className="px-3 pt-3 pb-1">
          <span className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
            Agents
          </span>
        </div>
        <div className="max-h-40 overflow-y-auto">
          <AgentSelector
            agents={agents}
            selected={selectedAgent}
            onSelect={handleSelectAgent}
            loading={agentsLoading}
          />
        </div>

        {/* Sessions section */}
        <div className="min-h-0 flex-1 overflow-y-auto border-t border-brand-shade3/15">
          {sessionsLoading ? (
            <div className="flex flex-col gap-2 p-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-10 animate-pulse rounded-btn bg-brand-shade3/10" />
              ))}
            </div>
          ) : (
            <SessionSidebar
              sessions={sessions}
              activeSessionId={activeSessionId}
              onSelectSession={handleSelectSession}
              onNewSession={handleNewSession}
              onRenameSession={handleRenameSession}
              onDeleteSession={handleDeleteSession}
            />
          )}
        </div>

        {/* Bottom nav */}
        <div className="border-t border-brand-shade3/15 p-3">
          <nav className="flex flex-col gap-1">
            <Link
              to="/agents"
              className="rounded-btn px-3 py-1.5 text-xs text-brand-shade3 transition-colors hover:bg-brand-shade3/10 hover:text-brand-light"
            >
              All Agents
            </Link>
            <Link
              to="/tasks"
              className="rounded-btn px-3 py-1.5 text-xs text-brand-shade3 transition-colors hover:bg-brand-shade3/10 hover:text-brand-light"
            >
              Tasks
            </Link>
            <Link
              to="/health"
              className="rounded-btn px-3 py-1.5 text-xs text-brand-shade3 transition-colors hover:bg-brand-shade3/10 hover:text-brand-light"
            >
              Health
            </Link>
          </nav>
        </div>
      </aside>

      {/* Main */}
      <main className="flex flex-1 flex-col">
        {/* Header */}
        <header className="flex items-center border-b border-brand-shade3/15 px-4 py-3 md:px-6">
          {/* Mobile hamburger */}
          <button
            className="mr-3 text-brand-shade3 transition-colors hover:text-brand-light md:hidden"
            onClick={() => setSidebarOpen(true)}
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </button>
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-brand-accent" />
            <span className="text-sm font-medium text-brand-light">
              {activeSession
                ? (activeSession.title || 'New chat')
                : selectedAgent
                  ? `${selectedAgent} - Select or create a session`
                  : 'Select an agent'}
            </span>
          </div>
          {messages.length > 0 && (
            <button
              onClick={handleClear}
              className="ml-auto text-xs text-brand-shade3 transition-colors hover:text-brand-light"
            >
              Clear chat
            </button>
          )}
        </header>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {loadingHistory ? (
            <div className="flex h-full items-center justify-center">
              <div className="flex items-center gap-2 text-brand-shade3">
                <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span className="text-sm">Loading messages...</span>
              </div>
            </div>
          ) : messages.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <div className="text-center">
                <p className="text-lg font-medium text-brand-shade3">
                  Start a conversation
                </p>
                <p className="mt-1 text-sm text-brand-shade3/60">
                  {selectedAgent
                    ? activeSessionId
                      ? `Send a message to ${selectedAgent}`
                      : 'Create a new session to start chatting'
                    : 'Select an agent from the sidebar'}
                </p>
              </div>
            </div>
          ) : (
            <div className="mx-auto flex max-w-5xl flex-col gap-3">
              {messages.map((msg) => (
                <ChatMessage key={msg.id} message={msg} onConfirmRespond={handleConfirmRespond} />
              ))}
              {streaming && messages.length > 0 && messages[messages.length - 1]?.role === 'user' && (
                <div className="flex justify-start animate-fade-in">
                  <ThinkingIndicator />
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* Input */}
        <div className="border-t border-brand-shade3/15 px-6 py-4">
          <div className="mx-auto max-w-5xl">
            <ChatInput
              onSend={handleSend}
              onStop={stop}
              streaming={streaming}
              disabled={!selectedAgent}
            />
          </div>
        </div>
      </main>
    </div>
  );
}
