// Test for action-based tool deduplication
import { render } from 'ink-testing-library';
import React from 'react';
import { describe, it, expect } from 'bun:test';
import { ToolGroupView } from '../ToolGroupView.js';
import { Message } from '../../../../domain/entities/Message.js';
import type { ChatMessage } from '../../../../domain/message.js';

// Message entity implements ChatMessage interface structurally
const toChatMessages = (msgs: Message[]) => msgs as unknown as ChatMessage[];

describe('ToolGroupView - action-based tool deduplication', () => {
  it('should deduplicate spawn_code_agent calls with same agent_id', () => {
    // Simulate: 14 status checks (running) → 1 final done
    const messages: Message[] = [];
    for (let i = 0; i < 14; i++) {
      messages.push(
        Message.createToolCall({
          callId: `tc${i}`,
          toolName: 'spawn_code_agent',
          arguments: { action: 'status', agent_id: 'code-agent-ed53c658' },
        }).markComplete().withToolResult('Status: running\nAgent ID: code-agent-ed53c658')
      );
    }
    messages.push(
      Message.createToolCall({
        callId: 'tc14',
        toolName: 'spawn_code_agent',
        arguments: { action: 'status', agent_id: 'code-agent-ed53c658' },
      }).markComplete().withToolResult('Status: done\nAgent ID: code-agent-ed53c658')
    );

    const { lastFrame } = render(<ToolGroupView messages={toChatMessages(messages)} />);
    const output = lastFrame();

    // Should only show ONE line (inline format, only last message = done)
    expect(output).toContain('● Agent');
    expect(output).toContain('→');
    expect(output).toContain('done');
    expect(output).toContain('ed53c658');

    // Should NOT show "running" (only last message = done)
    expect(output).not.toContain('running');

    // Should NOT show tree format (only 1 deduped message → inline)
    expect(output).not.toContain('└');
  });

  it('should deduplicate manage_subtasks calls with same subtask_id', () => {
    // Simulate: 5 list calls for same subtask → only last one shown
    const messages: Message[] = [];
    for (let i = 0; i < 5; i++) {
      messages.push(
        Message.createToolCall({
          callId: `tc${i}`,
          toolName: 'manage_subtasks',
          arguments: { action: 'list', task_id: 'task-abc', subtask_id: 'subtask-xyz' },
        }).markComplete().withToolResult(`Subtasks (${i + 1})`)
      );
    }

    const { lastFrame } = render(<ToolGroupView messages={toChatMessages(messages)} />);
    const output = lastFrame();

    // Should only show ONE line (inline, last message)
    expect(output).toContain('● Subtasks');
    expect(output).toContain('5 subtasks'); // Last result
    expect(output).toContain('subtask-xyz'); // keyArg

    // Should NOT show counts from earlier calls
    expect(output).not.toContain('1 subtask');
    expect(output).not.toContain('2 subtask');
  });

  it('should keep multiple messages with different keyArgs', () => {
    // Two different agents → should NOT deduplicate
    const msg1 = Message.createToolCall({
      callId: 'tc1',
      toolName: 'spawn_code_agent',
      arguments: { action: 'status', agent_id: 'agent-aaa' },
    }).markComplete().withToolResult('Status: running\nAgent ID: agent-aaa');

    const msg2 = Message.createToolCall({
      callId: 'tc2',
      toolName: 'spawn_code_agent',
      arguments: { action: 'status', agent_id: 'agent-bbb' },
    }).markComplete().withToolResult('Status: done\nAgent ID: agent-bbb');

    const { lastFrame } = render(<ToolGroupView messages={toChatMessages([msg1, msg2])} />);
    const output = lastFrame();

    // Should show BOTH lines (different keyArgs)
    expect(output).toContain('● Agent');
    expect(output).toContain('└'); // Tree format (2 lines)
    expect(output).toContain('agent-aaa');
    expect(output).toContain('agent-bbb');
  });

  it('should NOT deduplicate non action-based tools', () => {
    // Read file called 3 times with same path → should show ALL 3
    const messages = [
      Message.createToolCall({
        callId: 'tc1',
        toolName: 'read_file',
        arguments: { path: 'hello.go' },
      }).markComplete().withToolResult('package main\n// v1'),
      Message.createToolCall({
        callId: 'tc2',
        toolName: 'read_file',
        arguments: { path: 'hello.go' },
      }).markComplete().withToolResult('package main\n// v2'),
      Message.createToolCall({
        callId: 'tc3',
        toolName: 'read_file',
        arguments: { path: 'hello.go' },
      }).markComplete().withToolResult('package main\n// v3'),
    ];

    const { lastFrame } = render(<ToolGroupView messages={toChatMessages(messages)} />);
    const output = lastFrame();

    // Should show ALL 3 lines (no deduplication)
    const lineCount = (output!.match(/└/g) || []).length;
    expect(lineCount).toBe(3);
  });

  it('should deduplicate manage_tasks by title', () => {
    // Multiple status checks for same task (by title)
    const messages = [
      Message.createToolCall({
        callId: 'tc1',
        toolName: 'manage_tasks',
        arguments: { action: 'status', title: 'Create backend API' },
      }).markComplete().withToolResult('Task status: pending'),
      Message.createToolCall({
        callId: 'tc2',
        toolName: 'manage_tasks',
        arguments: { action: 'status', title: 'Create backend API' },
      }).markComplete().withToolResult('Task status: in_progress'),
      Message.createToolCall({
        callId: 'tc3',
        toolName: 'manage_tasks',
        arguments: { action: 'status', title: 'Create backend API' },
      }).markComplete().withToolResult('Task approved\nTitle: Create backend API'),
    ];

    const { lastFrame } = render(<ToolGroupView messages={toChatMessages(messages)} />);
    const output = lastFrame();

    // Should only show last message (approved)
    expect(output).toContain('● Tasks');
    expect(output).toContain('→'); // Inline format
    expect(output).toContain('approved');

    // Should NOT show earlier statuses
    expect(output).not.toContain('pending');
    expect(output).not.toContain('in_progress');
  });
});
