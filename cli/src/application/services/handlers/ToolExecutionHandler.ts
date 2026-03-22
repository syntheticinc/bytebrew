// ToolExecutionHandler - handles tool calls and execution
import { MessageId } from '../../../domain/value-objects/MessageId.js';
import { Message, ToolCallInfo, SubQuery } from '../../../domain/entities/Message.js';
import { ToolExecution } from '../../../domain/entities/ToolExecution.js';
import { StreamResponse, SubResult } from '../../../domain/ports/IStreamGateway.js';
import { StreamProcessorContext, completeCurrentMessage } from './StreamProcessorContext.js';
import { getLogger } from '../../../lib/logger.js';
import { debugLog } from '../../../lib/debugLog.js';

/**
 * Handle tool call response
 */
export function handleToolCall(ctx: StreamProcessorContext, response: StreamResponse): void {
  if (!response.toolCall) {
    return;
  }

  const toolCall: ToolCallInfo = {
    callId: response.toolCall.callId,
    toolName: response.toolCall.toolName,
    arguments: response.toolCall.arguments,
    subQueries: response.toolCall.subQueries,
    agentId: ctx.agentId,
  };

  const logger = getLogger();
  const isServerSideTool = toolCall.callId.startsWith('server-');
  logger.info('TOOL_CALL received', {
    callId: toolCall.callId,
    toolName: toolCall.toolName,
    isServerSide: isServerSideTool,
  });
  debugLog('TOOL_CALL', 'received', { callId: toolCall.callId, toolName: toolCall.toolName, isServerSide: isServerSideTool });

  // ask_user tool: save question as assistant message + user response as user message.
  // No tool message created (question is shown as regular chat, not as tool call).
  // Question is saved to history AFTER user responds (not before) to avoid duplication
  // with the interactive QuestionnairePrompt component.
  if (toolCall.toolName === 'ask_user') {
    completeCurrentMessage(ctx);

    // In display-only mode (no toolExecutor), publish AskUser event directly
    if (!ctx.toolExecutor) {
      ctx.eventBus.publish({
        type: 'AskUserRequested',
        questions: parseAskUserQuestions(toolCall.arguments),
        callId: toolCall.callId,
      });
      return;
    }

    executeAskUserAsync(ctx, toolCall).catch((err) => {
      ctx.eventBus.publish({
        type: 'ErrorOccurred',
        error: err as Error,
        context: `Tool background execution: ${toolCall.toolName}`,
      });
    });
    return;
  }

  // Check if message already exists for this callId (avoid duplicate TOOL_CALLs)
  const existingMessage = ctx.messageRepository.findByToolCallId(toolCall.callId);
  if (existingMessage) {
    logger.debug('TOOL_CALL duplicate skipped', { callId: toolCall.callId });
    return;
  }

  // Display-only mode (streaming API): no client-side execution, just show the tool call
  if (!ctx.toolExecutor) {
    completeCurrentMessage(ctx);

    const toolMessage = Message.createToolCall(toolCall, ctx.agentId);
    ctx.messageRepository.save(toolMessage);

    const execution = ToolExecution.create(
      toolCall.callId,
      toolCall.toolName,
      toolCall.arguments,
      toolMessage.id,
      ctx.agentId,
    );
    ctx.eventBus.publish({ type: 'ToolExecutionStarted', execution });
    // Result will arrive via TOOL_RESULT (handleServerToolResult)
    return;
  }

  // --- Legacy bidirectional mode below ---

  // For proxy TOOL_CALLs (non-server-), check if the server-side callback already
  // created a message with callId "server-{callId}". If so, reuse that message
  // to avoid showing the same tool call twice in the UI.
  if (!isServerSideTool) {
    const serverCallId = `server-${toolCall.callId}`;
    const serverMessage = ctx.messageRepository.findByToolCallId(serverCallId);
    if (serverMessage) {
      logger.debug('Proxy TOOL_CALL reuses server message', {
        proxyCallId: toolCall.callId,
        serverCallId,
      });
      // Start execution using the already-created server message
      const hasSubQueries = toolCall.subQueries && toolCall.subQueries.length > 0;
      const executor = hasSubQueries
        ? executeSubQueries(ctx, toolCall, serverMessage.id)
        : executeToolAsync(ctx, toolCall, serverMessage.id);

      executor.catch((err) => {
        ctx.eventBus.publish({
          type: 'ErrorOccurred',
          error: err as Error,
          context: `Tool background execution: ${toolCall.toolName}`,
        });
      });
      return;
    }
  }

  try {
    // Complete current message before tool call
    completeCurrentMessage(ctx);

    // Create tool message (immediately visible)
    const toolMessage = Message.createToolCall(toolCall, ctx.agentId);
    ctx.messageRepository.save(toolMessage);

    // Create tool execution entity
    const execution = ToolExecution.create(
      toolCall.callId,
      toolCall.toolName,
      toolCall.arguments,
      toolMessage.id,
      ctx.agentId
    );

    ctx.eventBus.publish({
      type: 'ToolExecutionStarted',
      execution,
    });

    // For client-side tools, execute in background
    // For server-side tools, wait for TOOL_RESULT from server
    if (!isServerSideTool) {
      const hasSubQueries = toolCall.subQueries && toolCall.subQueries.length > 0;
      const executor = hasSubQueries
        ? executeSubQueries(ctx, toolCall, toolMessage.id)
        : executeToolAsync(ctx, toolCall, toolMessage.id);

      executor.catch((err) => {
        ctx.eventBus.publish({
          type: 'ErrorOccurred',
          error: err as Error,
          context: `Tool background execution: ${toolCall.toolName}`,
        });
      });
    }
    // Server-side tools: message will be updated when TOOL_RESULT arrives
  } catch (error) {
    const err = error as Error;
    logger.error('TOOL_CALL sync setup failed', {
      callId: toolCall.callId,
      toolName: toolCall.toolName,
      error: err.message,
    });

    // If the synchronous setup failed for a client-side tool, send an error result
    // to the server so the proxy is not blocked indefinitely on resultChan.
    if (!isServerSideTool) {
      ctx.streamGateway.sendToolResult(
        toolCall.callId,
        `[ERROR] Tool setup failed: ${err.message}`,
        err
      );
    }
  }
}

/**
 * Execute tool asynchronously
 */
export async function executeToolAsync(
  ctx: StreamProcessorContext,
  toolCall: ToolCallInfo,
  messageId: MessageId
): Promise<void> {
  const logger = getLogger();
  logger.info('executeToolAsync start', { callId: toolCall.callId, toolName: toolCall.toolName });
  debugLog('EXEC', 'start', { callId: toolCall.callId, toolName: toolCall.toolName });
  try {
    const result = await ctx.toolExecutor.execute(toolCall);
    logger.info('executeToolAsync completed', { callId: toolCall.callId, toolName: toolCall.toolName, hasError: !!result.error });
    debugLog('EXEC', 'completed', { callId: toolCall.callId, hasError: !!result.error, resultLen: result.result?.length });

    // Update message with result
    const existingMessage = ctx.messageRepository.findById(messageId);
    if (existingMessage) {
      const updatedMessage = existingMessage.withToolResult(
        result.result,
        result.error?.message,
        result.diffLines
      );
      ctx.messageRepository.save(updatedMessage);

      ctx.eventBus.publish({
        type: 'MessageCompleted',
        message: updatedMessage,
      });
    }

    // Create completed execution
    const execution = ToolExecution.create(
      toolCall.callId,
      toolCall.toolName,
      toolCall.arguments,
      messageId,
      ctx.agentId
    );
    const completedExecution = result.error
      ? execution.fail(result.error.message)
      : execution.complete(result.result, result.summary, result.diffLines);

    ctx.eventBus.publish({
      type: 'ToolExecutionCompleted',
      execution: completedExecution,
    });

    // Send result back to server
    logger.info('Sending tool result to server', { callId: toolCall.callId, toolName: toolCall.toolName });
    debugLog('EXEC', 'Sending tool result to server', { callId: toolCall.callId, toolName: toolCall.toolName });
    ctx.streamGateway.sendToolResult(
      toolCall.callId,
      result.result,
      result.error
    );
  } catch (error) {
    const err = error as Error;
    logger.error('executeToolAsync failed', { callId: toolCall.callId, toolName: toolCall.toolName, error: err.message });
    debugLog('EXEC', 'FAILED', { callId: toolCall.callId, error: err.message });
    ctx.eventBus.publish({
      type: 'ErrorOccurred',
      error: err,
      context: `Tool execution: ${toolCall.toolName}`,
    });

    // Send error result back to server to prevent indefinite blocking.
    // Without this, the server's proxy waits for a result that never comes.
    ctx.streamGateway.sendToolResult(
      toolCall.callId,
      `[ERROR] ${err.message}`,
      err
    );
  }
}

/**
 * Execute ask_user tool and save user's response to chat history.
 * Unlike regular tools, ask_user doesn't create a tool message.
 * Instead: questions → assistant message, answers → user message.
 */
async function executeAskUserAsync(
  ctx: StreamProcessorContext,
  toolCall: ToolCallInfo
): Promise<void> {
  try {
    const result = await ctx.toolExecutor.execute(toolCall);

    // Format questions as assistant message (AFTER user responds to avoid duplication)
    const questionsStr = toolCall.arguments.questions || '[]';
    try {
      const questions = JSON.parse(questionsStr);
      if (Array.isArray(questions) && questions.length > 0) {
        const formatted = questions.map((q: any, i: number) => `${i + 1}. ${q.text}`).join('\n');
        const questionMessage = Message.createAssistantWithContent(formatted, ctx.agentId);
        ctx.messageRepository.save(questionMessage);
        ctx.eventBus.publish({ type: 'MessageCompleted', message: questionMessage });
      }
    } catch { /* ignore parse errors — legacy format handled below */ }

    // Format answers as user message
    if (result.result && !result.error) {
      try {
        const answers = JSON.parse(result.result);
        if (Array.isArray(answers)) {
          const formatted = answers.map((a: any) => `${a.question}: ${a.answer}`).join('\n');
          const userMessage = Message.createUser(formatted);
          ctx.messageRepository.save(userMessage);
          ctx.eventBus.publish({ type: 'MessageCompleted', message: userMessage });
        }
      } catch {
        // Fallback: save raw result (legacy single-answer format)
        if (result.result) {
          const userMessage = Message.createUser(result.result);
          ctx.messageRepository.save(userMessage);
          ctx.eventBus.publish({ type: 'MessageCompleted', message: userMessage });
        }
      }
    }

    // Create completed execution
    const messageId = MessageId.create();
    const execution = ToolExecution.create(
      toolCall.callId,
      toolCall.toolName,
      toolCall.arguments,
      messageId,
      ctx.agentId
    );
    const completedExecution = result.error
      ? execution.fail(result.error.message)
      : execution.complete(result.result, result.summary);

    ctx.eventBus.publish({
      type: 'ToolExecutionCompleted',
      execution: completedExecution,
    });

    // Send result back to server
    ctx.streamGateway.sendToolResult(
      toolCall.callId,
      result.result,
      result.error
    );
  } catch (error) {
    const err = error as Error;
    ctx.eventBus.publish({
      type: 'ErrorOccurred',
      error: err,
      context: `Tool execution: ${toolCall.toolName}`,
    });

    ctx.streamGateway.sendToolResult(
      toolCall.callId,
      `[ERROR] ${err.message}`,
      err
    );
  }
}

/**
 * Execute subQueries for grouped search operations (e.g. smart_search).
 * Creates UI messages for visibility, executes sub-queries in parallel,
 * updates message with summary, and sends results back to server.
 */
export async function executeSubQueries(
  ctx: StreamProcessorContext,
  toolCall: ToolCallInfo,
  messageId: MessageId
): Promise<void> {
  if (!toolCall.subQueries) {
    return;
  }

  try {
    // Map subQuery types to tool names
    const toolNameMap: Record<string, string> = {
      symbol: 'symbol_search',
      vector: 'search_code',
      grep: 'grep_search',
    };

    const SUB_QUERY_TIMEOUT_MS = 30_000;

    // Execute all subQueries in parallel with per-query timeout
    const subResultPromises = toolCall.subQueries.map(async (sq: SubQuery): Promise<SubResult> => {
      const toolName = toolNameMap[sq.type] || sq.type;

      // Build arguments based on query type
      const args: Record<string, string> = {};
      if (sq.type === 'grep') {
        args.pattern = buildGrepPattern(sq.query);
        args.limit = String(sq.limit);
        args.ignore_case = 'true'; // NL queries need case-insensitive matching
      } else if (sq.type === 'symbol') {
        const words = sq.query.split(/\s+/).filter(w => w.length > 2);
        args.symbol_name = words[0] || sq.query;
        args.limit = String(sq.limit);
      } else if (sq.type === 'vector') {
        args.query = sq.query;
        args.limit = String(sq.limit);
      }

      const subToolCall: ToolCallInfo = {
        callId: `${toolCall.callId}-${sq.type}`,
        toolName,
        arguments: args,
      };

      try {
        const result = await Promise.race([
          ctx.toolExecutor.execute(subToolCall),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error(`${toolName} timed out after ${SUB_QUERY_TIMEOUT_MS / 1000}s`)), SUB_QUERY_TIMEOUT_MS)
          ),
        ]);
        const count = countResults(result.result);
        return {
          type: sq.type,
          result: result.result,
          count,
          error: result.error?.message,
        };
      } catch (err) {
        return {
          type: sq.type,
          result: '',
          count: 0,
          error: (err as Error).message,
        };
      }
    });

    const subResults = await Promise.all(subResultPromises);

    // Symbol search fallback: if symbol returned 0, try grep for definitions
    await trySymbolFallback(ctx, toolCall, subResults);

    // Build formatted citations for SmartSearchRenderer
    const formattedResult = formatSubResultsAsCitations(subResults);

    // Build summary for progress tracking
    const summary = subResults
      .map(r => `${r.type}: ${r.error ? 'error' : r.count}`)
      .join(', ');

    // Update tool message with formatted result (for renderer) or summary as fallback
    const existingMessage = ctx.messageRepository.findById(messageId);
    if (existingMessage) {
      const updatedMessage = existingMessage.withToolResult(formattedResult || summary);
      ctx.messageRepository.save(updatedMessage);
      ctx.eventBus.publish({ type: 'MessageCompleted', message: updatedMessage });
    }

    // Create completed execution
    const execution = ToolExecution.create(
      toolCall.callId, toolCall.toolName, toolCall.arguments, messageId, ctx.agentId
    );
    ctx.eventBus.publish({
      type: 'ToolExecutionCompleted',
      execution: execution.complete('', summary),
    });

    // Send subResults back to server
    ctx.streamGateway.sendToolResult(
      toolCall.callId,
      '',
      undefined,
      subResults
    );
  } catch (error) {
    const err = error as Error;
    ctx.eventBus.publish({
      type: 'ErrorOccurred',
      error: err,
      context: `SubQueries execution: ${toolCall.toolName}`,
    });

    ctx.streamGateway.sendToolResult(toolCall.callId, '', err);
  }
}

interface CitationParseResult {
  citations: string[];
  nextIndex: number;
}

/**
 * Parse grep sub-result into numbered citations.
 * Grep format: "path:line\n  content\n\npath:line\n  content"
 */
function parseGrepCitations(result: string, startIndex: number): CitationParseResult {
  const citations: string[] = [];
  let index = startIndex;

  const entries = result.split('\n\n').filter(Boolean);
  for (const entry of entries) {
    const lines = entry.split('\n');
    if (lines.length >= 1) {
      const location = lines[0].trim();
      const preview = lines.length > 1 ? lines[1].trim() : '';
      if (location) {
        citations.push(`${index}. ${location} [grep] ${preview}`);
        index++;
      }
    }
  }

  return { citations, nextIndex: index };
}

/**
 * Parse vector sub-result into numbered citations.
 * Vector format: "## chunkType: name\nFile: path:start-end\nScore: N\n```...\ncontent\n```"
 */
function parseVectorCitations(result: string, startIndex: number): CitationParseResult {
  const citations: string[] = [];
  let index = startIndex;

  const chunks = result.split('## ').filter(Boolean);
  for (const chunk of chunks) {
    const lines = chunk.split('\n');
    // First line: "chunkType: name"
    const headerMatch = lines[0]?.match(/^(\w+):\s+(.+)/);
    // Find "File:" line
    const fileLine = lines.find(l => l.startsWith('File:'));
    if (fileLine) {
      const location = fileLine.replace('File: ', '').trim();
      const chunkType = headerMatch?.[1] || '';
      const name = headerMatch?.[2] || '';
      const info = chunkType ? `(${chunkType}) ${name}` : name;
      citations.push(`${index}. ${location} [vector] ${info}`);
      index++;
    }
  }

  return { citations, nextIndex: index };
}

/**
 * Parse symbol sub-result into numbered citations.
 * Symbol format: "[type] symbolName - signature\n  path:start-end"
 */
function parseSymbolCitations(result: string, startIndex: number): CitationParseResult {
  const citations: string[] = [];
  let index = startIndex;

  const entries = result.split('\n\n').filter(Boolean);
  for (const entry of entries) {
    const lines = entry.split('\n').map(l => l.trim()).filter(Boolean);
    // First line: "[type] symbolName - signature"
    const headerMatch = lines[0]?.match(/^\[(\w+)\]\s+(.+)/);
    // Second line: "path:start-end"
    const location = lines[1] || '';
    if (headerMatch && location) {
      const type = headerMatch[1];
      const symbolName = headerMatch[2].split(' - ')[0];
      citations.push(`${index}. ${location} [symbol] (${type}) ${symbolName}`);
      index++;
    }
  }

  return { citations, nextIndex: index };
}

/**
 * Convert raw sub-results from grep/vector/symbol searches into citation format
 * expected by SmartSearchRenderer: "N. location [source] info"
 *
 * Regex used by parseSmartSearchResult:
 *   line.match(/^\d+\.\s+(.+?)\s+\[(\w+)\](.*)$/)
 * Groups: (1) location, (2) source, (3) rest (optional symbol info)
 */
export function formatSubResultsAsCitations(subResults: SubResult[]): string {
  const citations: string[] = [];
  let index = 1;

  for (const sr of subResults) {
    if (sr.error || sr.count === 0 || !sr.result) continue;

    let parsed: CitationParseResult;
    if (sr.type === 'grep') {
      parsed = parseGrepCitations(sr.result, index);
    } else if (sr.type === 'vector') {
      parsed = parseVectorCitations(sr.result, index);
    } else if (sr.type === 'symbol') {
      parsed = parseSymbolCitations(sr.result, index);
    } else {
      continue;
    }

    citations.push(...parsed.citations);
    index = parsed.nextIndex;
  }

  return citations.join('\n');
}

/**
 * Count results in a search result string
 */
export function countResults(result: string): number {
  if (!result) return 0;
  // "No results/matches/symbols found" = 0 results
  if (/^No (results|matches|symbols) found/i.test(result.trim())) return 0;
  // Count entries separated by double newlines
  const entries = result.split('\n\n').filter((e) => e.trim());
  if (entries.length > 0) return entries.length;
  // Fallback: count non-empty lines
  return result.split('\n').filter((line) => line.trim()).length;
}

/**
 * Build grep pattern from a natural language query.
 *
 * Strategy:
 * - Code identifier (camelCase/PascalCase/snake_case): use as-is
 * - Multi-word: OR of all significant words (> 2 chars), case-insensitive grep
 *   handles matching variants (Request, request, etc.)
 */
export function buildGrepPattern(query: string): string {
  const trimmed = query.trim();

  // Code identifier — use as-is for exact matching
  if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(trimmed)) {
    return trimmed;
  }

  const words = trimmed.split(/\s+/).filter(w => w.length > 2);
  if (words.length <= 1) {
    return trimmed;
  }

  return words.join('|');
}

/**
 * When symbol search returns 0 results (no index), fall back to
 * grep for definition patterns: func/type/class/interface/struct + symbolName.
 * This gives tree-sitter-like results using just grep.
 */
async function trySymbolFallback(
  ctx: StreamProcessorContext,
  toolCall: ToolCallInfo,
  subResults: SubResult[]
): Promise<void> {
  const symbolResult = subResults.find(r => r.type === 'symbol');
  if (!symbolResult || symbolResult.count > 0) {
    return; // Symbol search found results, no fallback needed
  }

  // Extract symbol name from original sub-query
  const symbolQuery = toolCall.subQueries?.find(sq => sq.type === 'symbol');
  if (!symbolQuery) {
    return;
  }

  const words = symbolQuery.query.split(/\s+/).filter(w => w.length > 2);
  const symbolName = words[0] || symbolQuery.query;

  // Escape regex special chars in symbol name
  const escaped = symbolName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

  // Pattern matches function/type/class/interface/struct definitions
  // Includes Go method receiver syntax: func (r *Receiver) MethodName
  const defPattern = `(func|type|class|interface|struct|const|var)\\s+(\\([^)]*\\)\\s+)?${escaped}`;

  try {
    const defToolCall: ToolCallInfo = {
      callId: `${toolCall.callId}-def-grep`,
      toolName: 'grep_search',
      arguments: {
        pattern: defPattern,
        limit: String(symbolQuery.limit),
      },
    };

    const defResult = await ctx.toolExecutor.execute(defToolCall);
    const defCount = countResults(defResult.result);

    if (defCount > 0) {
      // Prepend definition results to grep channel (not symbol — format mismatch).
      // Server parseGrepResults handles this format. Definitions appear first
      // in grep results, giving them natural priority.
      const grepResult = subResults.find(r => r.type === 'grep');
      if (grepResult) {
        const separator = grepResult.result && grepResult.count > 0 ? '\n\n' : '';
        grepResult.result = defResult.result + separator + grepResult.result;
        grepResult.count += defCount;
      }
    }
  } catch {
    // Fallback failed silently — keep original empty result
  }
}

/**
 * Parse ask_user questions from tool arguments
 */
function parseAskUserQuestions(args: Record<string, string>): { text: string; options?: { label: string }[]; default?: string }[] {
  const questionsStr = args.questions;
  if (questionsStr) {
    try {
      const parsed = JSON.parse(questionsStr);
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed;
      }
    } catch { /* fall through */ }
  }
  const question = args.question || 'No question provided';
  return [{ text: question, default: args.default_answer }];
}

/**
 * Handle server-side tool result (e.g., smart_search)
 */
export function handleServerToolResult(ctx: StreamProcessorContext, response: StreamResponse): void {
  // Server tool results come in toolResult field
  if (!response.toolResult) {
    return; // No tool result, skip
  }

  const { callId, result, error, summary } = response.toolResult;

  // Extract tool name from callId (format: "server-{toolName}-{step}")
  const parts = callId.split('-');
  const toolName = parts.length >= 2 ? parts[1] : 'unknown';

  // Find existing message by callId
  const existingMessage = ctx.messageRepository.findByToolCallId(callId);

  if (existingMessage) {
    // Update existing message with result
    const updatedMessage = existingMessage.withToolResult(
      result,
      error
    );
    ctx.messageRepository.save(updatedMessage);

    ctx.eventBus.publish({
      type: 'MessageCompleted',
      message: updatedMessage,
    });

    // Create completed execution
    const execution = ToolExecution.create(
      callId,
      toolName,
      existingMessage.toolCall?.arguments || {},
      existingMessage.id,
      existingMessage.agentId
    );
    const completedExecution = error
      ? execution.fail(error)
      : execution.complete(result, summary);

    ctx.eventBus.publish({
      type: 'ToolExecutionCompleted',
      execution: completedExecution,
    });
  } else {
    // Fallback: no existing message found (shouldn't happen normally)
    // Create execution without message
    const execution = ToolExecution.create(
      callId,
      toolName,
      {},
      MessageId.create(),
      ctx.agentId
    );
    const completedExecution = error
      ? execution.fail(error)
      : execution.complete(result, summary);

    ctx.eventBus.publish({
      type: 'ToolExecutionCompleted',
      execution: completedExecution,
    });
  }
}
