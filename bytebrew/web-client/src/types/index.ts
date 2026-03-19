export interface AgentInfo {
  name: string;
  description?: string;
  tools_count: number;
  kit?: string;
  has_knowledge: boolean;
}

export interface AgentDetail extends AgentInfo {
  system_prompt: string;
  tools: string[];
  can_spawn: string[];
  lifecycle: string;
  max_steps: number;
  tool_execution?: string;
  max_context_size?: number;
}

export interface HealthResponse {
  status: string;
  version: string;
  uptime: string;
  agents_count: number;
  database?: string;
}

export type ChatEventType =
  | 'thinking'
  | 'message'
  | 'message_delta'
  | 'tool_call'
  | 'tool_result'
  | 'done'
  | 'error'
  | 'confirmation';

export interface ChatEvent {
  type: ChatEventType;
  data: Record<string, unknown>;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'thinking' | 'tool_call' | 'tool_result' | 'error' | 'confirmation' | 'info';
  content: string;
  toolName?: string;
  confirmArgs?: Record<string, unknown>;
  confirmPrompt?: string;
  timestamp: Date;
}

export interface PaginatedTaskResponse {
  data: TaskResponse[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface TaskResponse {
  id: number;
  title: string;
  agent_name: string;
  status: string;
  source: string;
  created_at: string;
}

export interface TaskDetailResponse extends TaskResponse {
  description: string;
  mode: string;
  result: string;
  error: string;
  started_at: string;
  completed_at: string;
}

export interface LoginResponse {
  token: string;
}
