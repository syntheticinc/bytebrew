import { describe, it, expect } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { ToolGroupView } from '../ToolGroupView.js';
import { ChatMessage, DiffLine } from '../../../../domain/message.js';

describe('ToolGroupView - Diff Display', () => {
  const tick = () => new Promise(r => setTimeout(r, 10));

  it('should display diff lines for single completed edit_file tool', async () => {
    const diffLines: DiffLine[] = [
      { type: ' ', content: 'function test() {' },
      { type: '-', content: '  console.log("old");' },
      { type: '+', content: '  console.log("new");' },
      { type: ' ', content: '}' },
    ];

    const messages: ChatMessage[] = [
      {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        timestamp: new Date(),
        isComplete: true,
        toolCall: {
          callId: 'call-1',
          toolName: 'edit_file',
          arguments: { file_path: 'test.ts', old_string: 'old', new_string: 'new' },
        },
        toolResult: {
          callId: 'call-1',
          toolName: 'edit_file',
          result: 'File updated successfully',
          summary: '+1 line',
          diffLines,
        },
      },
    ];

    const instance = render(<ToolGroupView messages={messages} />);
    await tick();

    const output = instance.lastFrame();

    // Check header
    expect(output).toContain('● Edit');
    expect(output).toContain('+1 line');

    // Check diff lines with correct colors (ink-testing-library doesn't preserve ANSI, check content)
    // Format: "  - " for removed, "  + " for added, "    " for context
    expect(output).toContain('function test() {');
    expect(output).toContain('-   console.log("old");');
    expect(output).toContain('+   console.log("new");');
    expect(output).toContain('}');

    instance.unmount();
  });

  it('should display diff lines in multi-tool expanded format', async () => {
    const diffLines1: DiffLine[] = [
      { type: '+', content: 'new line 1' },
    ];
    const diffLines2: DiffLine[] = [
      { type: '-', content: 'deleted line' },
      { type: '+', content: 'added line' },
    ];

    const messages: ChatMessage[] = [
      {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        timestamp: new Date(),
        isComplete: true,
        toolCall: {
          callId: 'call-1',
          toolName: 'write_file',
          arguments: { file_path: 'file1.ts' },
        },
        toolResult: {
          callId: 'call-1',
          toolName: 'write_file',
          result: 'Created',
          summary: '+1 line',
          diffLines: diffLines1,
        },
      },
      {
        id: 'msg-2',
        role: 'assistant',
        content: '',
        timestamp: new Date(),
        isComplete: true,
        toolCall: {
          callId: 'call-2',
          toolName: 'write_file',
          arguments: { file_path: 'file2.ts' },
        },
        toolResult: {
          callId: 'call-2',
          toolName: 'write_file',
          result: 'Created',
          summary: '+1 -1 lines',
          diffLines: diffLines2,
        },
      },
    ];

    const instance = render(<ToolGroupView messages={messages} />);
    await tick();

    const output = instance.lastFrame();

    // Check both diffs appear (with leading spaces)
    expect(output).toContain('+ new line 1');
    expect(output).toContain('- deleted line');
    expect(output).toContain('+ added line');

    instance.unmount();
  });

  it('should NOT display diff section if diffLines is empty', async () => {
    const messages: ChatMessage[] = [
      {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        timestamp: new Date(),
        isComplete: true,
        toolCall: {
          callId: 'call-1',
          toolName: 'edit_file',
          arguments: { file_path: 'test.ts' },
        },
        toolResult: {
          callId: 'call-1',
          toolName: 'edit_file',
          result: 'No changes',
          summary: 'no changes',
          diffLines: [], // Empty
        },
      },
    ];

    const instance = render(<ToolGroupView messages={messages} />);
    await tick();

    const output = instance.lastFrame();

    // Should have header but no diff lines
    expect(output).toContain('● Edit');
    expect(output).toContain('no changes');
    // No '+' or '-' prefix should appear (since no diff)
    expect(output).not.toMatch(/\+\s+\w/);
    expect(output).not.toMatch(/-\s+\w/);

    instance.unmount();
  });

  it('should NOT display diff section if diffLines is undefined', async () => {
    const messages: ChatMessage[] = [
      {
        id: 'msg-1',
        role: 'assistant',
        content: '',
        timestamp: new Date(),
        isComplete: true,
        toolCall: {
          callId: 'call-1',
          toolName: 'read_file',
          arguments: { file_path: 'test.ts' },
        },
        toolResult: {
          callId: 'call-1',
          toolName: 'read_file',
          result: '50 lines',
          summary: '50 lines',
          // diffLines is undefined (read_file doesn't produce diffs)
        },
      },
    ];

    const instance = render(<ToolGroupView messages={messages} />);
    await tick();

    const output = instance.lastFrame();

    // Should have header but no diff
    expect(output).toContain('● Read');
    expect(output).toContain('50 lines');
    expect(output).not.toContain('+');
    expect(output).not.toContain('-');

    instance.unmount();
  });
});
