// Test for inline format (single completed tool call)
import { render } from 'ink-testing-library';
import React from 'react';
import { describe, it, expect } from 'bun:test';
import { ToolGroupView } from '../ToolGroupView.js';
import { Message } from '../../../../domain/entities/Message.js';
import { toMessageViewModel } from '../../../mappers/MessageViewMapper.js';

describe('ToolGroupView - inline format', () => {
  it('should render single completed call as inline (one line)', () => {
    const msg = Message.createToolCall({
      callId: 'tc1',
      toolName: 'manage_tasks',
      arguments: { action: 'create', title: 'Create hello.go' },
    }).markComplete().withToolResult('Task created successfully\nTitle: Create hello.go');

    const { lastFrame } = render(<ToolGroupView messages={[toMessageViewModel(msg)]} />);
    const output = lastFrame();

    // Should contain inline format with arrow
    expect(output).toContain('● Tasks');
    expect(output).toContain('→');
    expect(output).toContain('created');
    expect(output).toContain('Create hello.go');

    // Should NOT contain tree-like format
    expect(output).not.toContain('└');
  });

  it('should render multiple calls as two-line format (multi-line)', () => {
    const msg1 = Message.createToolCall({
      callId: 'tc1',
      toolName: 'manage_tasks',
      arguments: { action: 'create', title: 'Task A' },
    }).markComplete().withToolResult('Task created successfully\nTitle: Task A');

    const msg2 = Message.createToolCall({
      callId: 'tc2',
      toolName: 'manage_tasks',
      arguments: { action: 'list' },
    }).markComplete().withToolResult('Tasks (2)');

    const { lastFrame } = render(<ToolGroupView messages={[toMessageViewModel(msg1), toMessageViewModel(msg2)]} />);
    const output = lastFrame();

    // Should contain two-line format
    expect(output).toContain('● Tasks');
    expect(output).toContain('└');

    // Should NOT contain inline arrow
    expect(output).not.toContain('→');
  });

  it('should deduplicate keyArg if already in summary', () => {
    const msg = Message.createToolCall({
      callId: 'tc1',
      toolName: 'read_file',
      arguments: { path: 'hello.go' },
    }).markComplete().withToolResult('', 'file not found: hello.go');

    const { lastFrame } = render(<ToolGroupView messages={[toMessageViewModel(msg)]} />);
    const output = lastFrame()!;

    // Should show error message
    expect(output).toContain('file not found');

    // Should NOT duplicate "hello.go" (appears in error, should not appear in keyArg)
    const occurrences = (output.match(/hello\.go/gi) || []).length;
    expect(occurrences).toBe(1);
  });

  it('should show keyArg if NOT in summary', () => {
    const msg = Message.createToolCall({
      callId: 'tc1',
      toolName: 'write_file',
      arguments: { path: 'output.txt', content: 'test' },
    }).markComplete().withToolResult('File written: output.txt (5 lines)');

    const { lastFrame } = render(<ToolGroupView messages={[toMessageViewModel(msg)]} />);
    const output = lastFrame()!;

    // formatResultSummary returns "5 lines" for write (extracts line count)
    expect(output).toContain('5 lines');

    // keyArg is "output.txt", summary is "5 lines" — should show keyArg
    expect(output).toContain('output.txt');
  });

  it('should use marginBottom=0 for inline format (compact)', () => {
    const msg = Message.createToolCall({
      callId: 'tc1',
      toolName: 'manage_tasks',
      arguments: { action: 'create' },
    }).markComplete().withToolResult('Task created');

    const vm = toMessageViewModel(msg);
    const { lastFrame } = render(
      <>
        <ToolGroupView messages={[vm]} />
        <ToolGroupView messages={[vm]} />
      </>
    );

    const output = lastFrame()!;

    // With marginBottom=0, lines should be adjacent (no empty line between)
    const lines = output.split('\n');
    const taskLines = lines.filter(l => l.includes('● Tasks'));

    // Should have 2 task lines close together
    expect(taskLines.length).toBe(2);
  });
});
