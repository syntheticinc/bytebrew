import { useState, useEffect, useRef, useCallback } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { useChat } from '../hooks/useChat';
import { useAuth } from '../hooks/useAuth';
import { ChatMessage } from '../components/ChatMessage';
import { ChatInput } from '../components/ChatInput';
import { AgentSelector } from '../components/AgentSelector';
import { ThinkingIndicator } from '../components/ThinkingIndicator';
import type { AgentInfo } from '../types';

export function ChatPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { logout } = useAuth();
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [agentsLoading, setAgentsLoading] = useState(true);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(
    searchParams.get('agent') || sessionStorage.getItem('bytebrew_last_agent'),
  );
  const { messages, streaming, send, respond, stop, clear } = useChat(selectedAgent);
  const messagesEndRef = useRef<HTMLDivElement>(null);

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

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSelectAgent = useCallback(
    (name: string) => {
      setSelectedAgent(name);
      setSearchParams({ agent: name });
      sessionStorage.setItem('bytebrew_last_agent', name);
    },
    [setSearchParams],
  );

  const handleSend = useCallback(
    (text: string) => {
      if (!selectedAgent) return;
      send(selectedAgent, text);
    },
    [selectedAgent, send],
  );

  const handleConfirmRespond = useCallback(
    (answer: 'yes' | 'no') => {
      if (!selectedAgent) return;
      respond(selectedAgent, answer);
    },
    [selectedAgent, respond],
  );

  return (
    <div className="flex h-screen bg-brand-dark">
      {/* Sidebar */}
      <aside className="flex w-56 flex-shrink-0 flex-col border-r border-brand-shade3/15 bg-brand-dark">
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

        <div className="px-3 pt-3 pb-1">
          <span className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
            Agents
          </span>
        </div>

        <div className="flex-1 overflow-y-auto">
          <AgentSelector
            agents={agents}
            selected={selectedAgent}
            onSelect={handleSelectAgent}
            loading={agentsLoading}
          />
        </div>

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
        <header className="flex items-center border-b border-brand-shade3/15 px-6 py-3">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-brand-accent" />
            <span className="text-sm font-medium text-brand-light">
              {selectedAgent ?? 'Select an agent'}
            </span>
          </div>
          {messages.length > 0 && (
            <button
              onClick={clear}
              className="ml-auto text-xs text-brand-shade3 transition-colors hover:text-brand-light"
            >
              Clear chat
            </button>
          )}
        </header>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {messages.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <div className="text-center">
                <p className="text-lg font-medium text-brand-shade3">
                  Start a conversation
                </p>
                <p className="mt-1 text-sm text-brand-shade3/60">
                  {selectedAgent
                    ? `Send a message to ${selectedAgent}`
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
