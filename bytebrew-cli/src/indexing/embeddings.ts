// Ollama embeddings client
import { IEmbeddingsClient } from '../domain/store.js';
import { getLogger } from '../lib/logger.js';

const DEFAULT_OLLAMA_URL = 'http://localhost:11434';
const DEFAULT_MODEL = 'nomic-embed-text';
const DEFAULT_DIMENSION = 768;
const DEFAULT_TIMEOUT = 60000; // 60 seconds
const DEFAULT_BATCH_SIZE = 50;
const MAX_RETRIES = 3;
const RETRY_DELAY_MS = 1000;
// nomic-embed-text context: 8192 tokens. ~4 chars/token → 32k chars.
// Use conservative limit as safety net when Ollama truncate: true doesn't work.
const MAX_EMBED_CHARS = 28000;

export interface EmbeddingsConfig {
  baseUrl?: string;
  model?: string;
  dimension?: number;
  timeout?: number;
  batchSize?: number;
}

// Ollama /api/embed request (newer endpoint with truncate support)
interface OllamaEmbedRequest {
  model: string;
  input: string | string[];
  truncate?: boolean;
}

// Ollama /api/embed response
interface OllamaEmbedResponse {
  model: string;
  embeddings: number[][];
}

export class EmbeddingsClient implements IEmbeddingsClient {
  private baseUrl: string;
  private model: string;
  private dimension: number;
  private timeout: number;
  private batchSize: number;

  constructor(config: EmbeddingsConfig = {}) {
    this.baseUrl = config.baseUrl || DEFAULT_OLLAMA_URL;
    this.model = config.model || DEFAULT_MODEL;
    this.dimension = config.dimension || DEFAULT_DIMENSION;
    this.timeout = config.timeout || DEFAULT_TIMEOUT;
    this.batchSize = config.batchSize || DEFAULT_BATCH_SIZE;
  }

  async embed(text: string): Promise<number[]> {
    const logger = getLogger();
    let lastError: Error | null = null;

    // Client-side truncation as safety net — some Ollama versions ignore truncate: true
    const truncated = text.length > MAX_EMBED_CHARS ? text.slice(0, MAX_EMBED_CHARS) : text;

    for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), this.timeout);

      try {
        const request: OllamaEmbedRequest = {
          model: this.model,
          input: truncated,
          truncate: true,
        };

        const response = await fetch(`${this.baseUrl}/api/embed`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(request),
          signal: controller.signal,
        });

        if (!response.ok) {
          const body = await response.text();
          const err = new Error(`Ollama API error: status ${response.status}, body: ${body}`);
          // 400 = bad request, retrying won't help
          if (response.status === 400) throw err;
          throw err;
        }

        const data = (await response.json()) as OllamaEmbedResponse;
        if (!data.embeddings || data.embeddings.length === 0) {
          throw new Error('Ollama returned empty embeddings');
        }
        return data.embeddings[0];
      } catch (error) {
        lastError = error as Error;
        clearTimeout(timeoutId);

        // Don't retry client errors (400) — input is bad, retrying won't fix it
        if (lastError.message.includes('status 400')) break;

        if (attempt < MAX_RETRIES) {
          logger.warn(`Embedding attempt ${attempt} failed, retrying...`, {
            error: lastError.message,
          });
          await this.delay(RETRY_DELAY_MS * attempt);
        }
      } finally {
        clearTimeout(timeoutId);
      }
    }

    logger.warn('Embedding failed, skipping chunk', { error: lastError?.message, textLength: text.length });
    throw lastError;
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async embedBatch(texts: string[]): Promise<(number[] | null)[]> {
    if (texts.length === 0) {
      return [];
    }

    const logger = getLogger();
    const results: (number[] | null)[] = [];

    for (let i = 0; i < texts.length; i += this.batchSize) {
      const batch = texts.slice(i, i + this.batchSize);

      // Try native batch API first (1 HTTP request for entire batch)
      const batchResult = await this.embedBatchNative(batch);
      if (batchResult) {
        results.push(...batchResult);
        continue;
      }

      // Fallback: embed individually (handles per-item failures)
      logger.debug('Batch embed failed, falling back to individual requests');
      for (const text of batch) {
        try {
          results.push(await this.embed(text));
        } catch {
          results.push(null);
        }
      }
    }

    return results;
  }

  /**
   * Send all texts in one Ollama /api/embed request (input: string[]).
   * Returns null if the batch request itself fails (caller should fallback).
   */
  private async embedBatchNative(texts: string[]): Promise<(number[] | null)[] | null> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout * 2);

    try {
      const truncated = texts.map(t =>
        t.length > MAX_EMBED_CHARS ? t.slice(0, MAX_EMBED_CHARS) : t
      );

      const request: OllamaEmbedRequest = {
        model: this.model,
        input: truncated,
        truncate: true,
      };

      const response = await fetch(`${this.baseUrl}/api/embed`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
        signal: controller.signal,
      });

      if (!response.ok) {
        return null; // Let caller fallback to individual
      }

      const data = (await response.json()) as OllamaEmbedResponse;
      if (!data.embeddings || data.embeddings.length !== texts.length) {
        return null;
      }

      return data.embeddings;
    } catch {
      return null;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  async ping(): Promise<boolean> {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 5000);

      const response = await fetch(`${this.baseUrl}/api/tags`, {
        method: 'GET',
        signal: controller.signal,
      });

      clearTimeout(timeoutId);
      return response.ok;
    } catch {
      return false;
    }
  }

  getDimension(): number {
    return this.dimension;
  }

  getModel(): string {
    return this.model;
  }
}
