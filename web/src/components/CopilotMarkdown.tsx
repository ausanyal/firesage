import React from 'react';

interface CopilotMarkdownProps {
  markdown: string;
}

function renderInline(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];
  // Tokenize inline code (`...`), bold (**...**), and italic (*...*)
  const tokenRe = /(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*)/g;
  let lastIdx = 0;
  let match: RegExpExecArray | null;
  let key = 0;

  while ((match = tokenRe.exec(text)) !== null) {
    if (match.index > lastIdx) {
      nodes.push(text.slice(lastIdx, match.index));
    }
    const tok = match[0];
    if (tok.startsWith('`') && tok.endsWith('`')) {
      nodes.push(
        <code key={key++} className="fs-md-code">
          {tok.slice(1, -1)}
        </code>,
      );
    } else if (tok.startsWith('**') && tok.endsWith('**')) {
      nodes.push(<strong key={key++}>{tok.slice(2, -2)}</strong>);
    } else if (tok.startsWith('*') && tok.endsWith('*')) {
      nodes.push(<em key={key++}>{tok.slice(1, -1)}</em>);
    } else {
      nodes.push(tok);
    }
    lastIdx = match.index + tok.length;
  }
  if (lastIdx < text.length) {
    nodes.push(text.slice(lastIdx));
  }
  return nodes;
}

function parseTableCells(line: string): string[] {
  const trimmed = line.trim().replace(/^\|/, '').replace(/\|$/, '');
  return trimmed.split('|').map((c) => c.trim());
}

function isTableSeparator(line: string): boolean {
  const t = line.trim();
  return t.startsWith('|') && /^[\s|:-]+$/.test(t) && t.includes('-');
}

export const CopilotMarkdown: React.FC<CopilotMarkdownProps> = ({ markdown }) => {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n');
  const blocks: React.ReactNode[] = [];
  let i = 0;
  let blockKey = 0;

  while (i < lines.length) {
    const raw = lines[i];
    const trimmed = raw.trim();

    if (!trimmed) {
      i++;
      continue;
    }

    // GFM Pipe Table
    if (trimmed.startsWith('|') && i + 1 < lines.length && isTableSeparator(lines[i + 1])) {
      const headers = parseTableCells(trimmed);
      i += 2; // Skip header & separator
      const rows: string[][] = [];
      while (i < lines.length && lines[i].trim().startsWith('|')) {
        if (!isTableSeparator(lines[i])) {
          rows.push(parseTableCells(lines[i]));
        }
        i++;
      }
      blocks.push(
        <div key={blockKey++} className="fs-table-scroll fs-md-table-wrap">
          <table className="fs-copilot-table">
            <caption className="visually-hidden">Quantitative retirement comparison metrics</caption>
            <thead>
              <tr>
                {headers.map((h, hIdx) => (
                  <th key={hIdx} scope="col">
                    {renderInline(h)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rIdx) => (
                <tr key={rIdx}>
                  {row.map((cell, cIdx) =>
                    cIdx === 0 ? (
                      <th key={cIdx} scope="row">
                        {renderInline(cell)}
                      </th>
                    ) : (
                      <td key={cIdx}>{renderInline(cell)}</td>
                    ),
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }

    // Headings (mapped inside Copilot drawer under h2 -> h3 -> h4 so hierarchy never skips)
    if (trimmed.startsWith('#### ')) {
      blocks.push(
        <h4 key={blockKey++} className="fs-md-h4">
          {renderInline(trimmed.slice(5))}
        </h4>,
      );
      i++;
      continue;
    }
    if (trimmed.startsWith('### ') || trimmed.startsWith('## ')) {
      const content = trimmed.replace(/^#{2,3}\s+/, '');
      blocks.push(
        <h3 key={blockKey++} className="fs-md-h3">
          {renderInline(content)}
        </h3>,
      );
      i++;
      continue;
    }

    // Blockquote
    if (trimmed.startsWith('> ')) {
      const quoteLines: string[] = [];
      while (i < lines.length && lines[i].trim().startsWith('>')) {
        quoteLines.push(lines[i].trim().replace(/^>\s?/, ''));
        i++;
      }
      blocks.push(
        <blockquote key={blockKey++} className="fs-md-quote">
          {renderInline(quoteLines.join(' '))}
        </blockquote>,
      );
      continue;
    }

    // Unordered list
    if (/^[-*•]\s+/.test(trimmed)) {
      const items: string[] = [];
      while (i < lines.length && /^[-*•]\s+/.test(lines[i].trim())) {
        items.push(lines[i].trim().replace(/^[-*•]\s+/, ''));
        i++;
      }
      blocks.push(
        <ul key={blockKey++} className="fs-md-list">
          {items.map((it, idx) => (
            <li key={idx}>{renderInline(it)}</li>
          ))}
        </ul>,
      );
      continue;
    }

    // Ordered list
    if (/^\d+\.\s+/.test(trimmed)) {
      const items: string[] = [];
      while (i < lines.length && /^\d+\.\s+/.test(lines[i].trim())) {
        items.push(lines[i].trim().replace(/^\d+\.\s+/, ''));
        i++;
      }
      blocks.push(
        <ol key={blockKey++} className="fs-md-list">
          {items.map((it, idx) => (
            <li key={idx}>{renderInline(it)}</li>
          ))}
        </ol>,
      );
      continue;
    }

    // Standard paragraph (join consecutive plain lines)
    const paraLines: string[] = [];
    while (
      i < lines.length &&
      lines[i].trim() !== '' &&
      !lines[i].trim().startsWith('#') &&
      !lines[i].trim().startsWith('|') &&
      !lines[i].trim().startsWith('>') &&
      !/^[-*•]\s+/.test(lines[i].trim()) &&
      !/^\d+\.\s+/.test(lines[i].trim())
    ) {
      paraLines.push(lines[i].trim());
      i++;
    }
    blocks.push(
      <p key={blockKey++} className="fs-md-p">
        {renderInline(paraLines.join(' '))}
      </p>,
    );
  }

  return <div className="fs-md-root">{blocks}</div>;
};
