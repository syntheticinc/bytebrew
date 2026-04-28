/**
 * Basic markdown to HTML converter.
 * Supports: bold, inline code, code blocks, links, lists, newlines.
 */
export function renderMarkdown(text: string): string {
  // Escape HTML entities first
  let html = escapeHtml(text);

  // Code blocks: ```...```
  html = html.replace(/```(\w*)\n?([\s\S]*?)```/g, (_match, _lang, code) => {
    return `<pre><code>${code.trim()}</code></pre>`;
  });

  // Inline code: `...`
  html = html.replace(/`([^`\n]+)`/g, '<code>$1</code>');

  // Bold: **...** or __...__
  html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  html = html.replace(/__([^_]+)__/g, '<strong>$1</strong>');

  // Links: [text](url)
  html = html.replace(
    /\[([^\]]+)\]\(([^)]+)\)/g,
    '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>',
  );

  // Process lines for lists and paragraphs
  html = processLines(html);

  return html;
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function processLines(html: string): string {
  const lines = html.split('\n');
  const result: string[] = [];
  let inList = false;

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines inside <pre> blocks (already handled)
    if (trimmed.startsWith('<pre>') || trimmed.endsWith('</pre>')) {
      if (inList) {
        result.push('</ul>');
        inList = false;
      }
      result.push(line);
      continue;
    }

    // List items: - item or * item
    if (/^[-*]\s+/.test(trimmed)) {
      if (!inList) {
        result.push('<ul>');
        inList = true;
      }
      result.push(`<li>${trimmed.replace(/^[-*]\s+/, '')}</li>`);
      continue;
    }

    // Close list if we're no longer in one
    if (inList) {
      result.push('</ul>');
      inList = false;
    }

    // Empty line
    if (trimmed === '') {
      continue;
    }

    // Regular line
    result.push(line + '<br>');
  }

  if (inList) {
    result.push('</ul>');
  }

  // Remove trailing <br>
  const joined = result.join('\n');
  return joined.replace(/<br>\s*$/, '');
}
