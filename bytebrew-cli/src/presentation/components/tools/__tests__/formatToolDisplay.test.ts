import { describe, it, expect } from 'bun:test';
import { formatResultSummary } from '../formatToolDisplay.js';

describe('formatResultSummary - server-provided summary', () => {
  it('should use server-provided summary if available', () => {
    const result = 'some long result text...';
    const summary = 'plan created';

    expect(formatResultSummary('manage_plan', result, undefined, summary)).toBe('plan created');
  });

  it('should fallback to client parsing if no summary', () => {
    const treeResult = `project/
├── src/
└── package.json`;

    expect(formatResultSummary('get_project_tree', treeResult, undefined)).toBe('2 items');
  });

  it('should show error if provided, ignoring summary', () => {
    const error = 'Something went wrong';
    const summary = 'plan created';

    expect(formatResultSummary('manage_plan', '', error, summary)).toBe(error);
  });
});

describe('formatResultSummary - tree items count', () => {
  it('should count tree entries (├── and └──)', () => {
    const treeResult = `project/
├── src/
│   ├── main.ts
│   ├── utils/
│   │   └── helper.ts
│   └── index.ts
├── package.json
└── README.md`;

    const summary = formatResultSummary('get_project_tree', treeResult);
    // Should count 7 entries with ├── or └──
    expect(summary).toBe('7 items');
  });

  it('should not count depth limit markers', () => {
    const treeResult = `project/
├── src/
│   ├── main.ts
│   └── (depth limit reached)
└── package.json`;

    const summary = formatResultSummary('get_project_tree', treeResult);
    // Should count 3 entries (src/, main.ts, package.json) — not depth limit line
    expect(summary).toBe('3 items');
  });

  it('should handle single item', () => {
    const treeResult = `project/
└── README.md`;

    const summary = formatResultSummary('get_project_tree', treeResult);
    expect(summary).toBe('1 item');
  });

  it('should count JSON tree nodes', () => {
    const jsonResult = JSON.stringify({
      name: 'project',
      is_directory: true,
      children: [
        { name: 'src', is_directory: true, children: [
          { name: 'main.ts', is_directory: false },
          { name: 'index.ts', is_directory: false },
        ]},
        { name: 'package.json', is_directory: false },
      ],
    });

    const summary = formatResultSummary('get_project_tree', jsonResult);
    // 3 children total: src (with 2 children) + package.json = 3 + 2 = 5? No: src, main.ts, index.ts, package.json = 4
    expect(summary).toBe('4 items');
  });

  it('should handle JSON tree with empty children', () => {
    const jsonResult = JSON.stringify({
      name: 'project',
      is_directory: true,
      children: [],
    });

    const summary = formatResultSummary('get_project_tree', jsonResult);
    // No children — fallback to non-empty lines
    expect(summary).toBe('1 item');
  });

  it('should work for list_dir tool too', () => {
    const listResult = `├── file1.ts
├── file2.ts
└── file3.ts`;

    const summary = formatResultSummary('list_dir', listResult);
    expect(summary).toBe('3 items');
  });
});
