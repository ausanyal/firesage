import type {
  ChatRequest,
  ChatResponse,
  FullAnalysisResponse,
  MetadataResponse,
  PlanInput,
} from '../types';

export type ExecutionBackend = 'go-cloud-api' | 'go-local-wasm';

let detectedBackend: ExecutionBackend | null = null;
let workerInstance: Worker | null = null;
let nextReqId = 1;
const pendingWorkerReqs = new Map<
  number,
  { resolve: (val: unknown) => void; reject: (err: Error) => void }
>();

function getWasmWorker(): Worker {
  if (!workerInstance) {
    workerInstance = new Worker(new URL('./wasmWorker.ts', import.meta.url), {
      type: 'module',
    });
    workerInstance.addEventListener('message', (ev) => {
      const { id, ok, data, error } = ev.data;
      const entry = pendingWorkerReqs.get(id);
      if (!entry) return;
      pendingWorkerReqs.delete(id);
      if (ok) {
        entry.resolve(data);
      } else {
        entry.reject(new Error(error || 'WASM worker error'));
      }
    });
  }
  return workerInstance;
}

function callWasmWorker<T>(type: 'metadata' | 'analyze' | 'chat', payload?: string): Promise<T> {
  const worker = getWasmWorker();
  const id = nextReqId++;
  return new Promise<T>((resolve, reject) => {
    pendingWorkerReqs.set(id, {
      resolve: (val) => resolve(val as T),
      reject,
    });
    worker.postMessage({ id, type, payload });
  });
}

async function detectBackend(): Promise<ExecutionBackend> {
  if (detectedBackend) return detectedBackend;
  try {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 500);
    const resp = await fetch('/api/v1/health', { signal: ctrl.signal });
    clearTimeout(timer);
    if (resp.ok) {
      const body = await resp.json();
      if (body && body.service === 'firesage') {
        detectedBackend = 'go-cloud-api';
        return detectedBackend;
      }
    }
  } catch {
    // Native Go HTTP server not active (e.g. offline Local-First or Vite static preview); use Go WASM worker
  }
  detectedBackend = 'go-local-wasm';
  return detectedBackend;
}

export async function fetchMetadata(): Promise<{ data: MetadataResponse; backend: ExecutionBackend }> {
  const backend = await detectBackend();
  if (backend === 'go-cloud-api') {
    const resp = await fetch('/api/v1/metadata');
    if (resp.ok) {
      return { data: (await resp.json()) as MetadataResponse, backend };
    }
    detectedBackend = 'go-local-wasm';
  }
  const data = await callWasmWorker<MetadataResponse>('metadata');
  return { data, backend: 'go-local-wasm' };
}

export async function runSimulationAnalysis(
  input: PlanInput,
): Promise<{ result: FullAnalysisResponse; backend: ExecutionBackend }> {
  const backend = await detectBackend();
  if (backend === 'go-cloud-api') {
    const resp = await fetch('/api/v1/analyze', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    if (resp.ok) {
      return { result: (await resp.json()) as FullAnalysisResponse, backend };
    }
  }

  // In single-threaded browser WASM mode, cap interactive slider-drag path count to 1,200
  // unless user explicitly selected Historical mode so UI updates stay snappy (<400ms).
  const tunedInput: PlanInput =
    input.mode !== 'historical' && input.num_paths > 1500
      ? { ...input, num_paths: 1200 }
      : input;

  const result = await callWasmWorker<FullAnalysisResponse>('analyze', JSON.stringify(tunedInput));
  return { result, backend: 'go-local-wasm' };
}

export async function runCopilotChat(
  req: ChatRequest,
): Promise<{ result: ChatResponse; backend: ExecutionBackend }> {
  const backend = await detectBackend();
  if (backend === 'go-cloud-api') {
    const resp = await fetch('/api/v1/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (resp.ok) {
      return { result: (await resp.json()) as ChatResponse, backend };
    }
  }

  // Offline / Local-First WASM Planner + Parallel Quantitative Tool Execution
  const wasmResult = await callWasmWorker<ChatResponse>('chat', JSON.stringify(req));

  // If user configured an optional Gemini API Key while running in pure Local-First WASM mode,
  // run Stage 3 Deep Reasoner directly from the browser over the WASM EvidencePack!
  const apiKey = (req.api_key || '').trim();
  if (apiKey) {
    try {
      const model = (req.model || 'gemini-2.5-flash').trim();
      const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/${encodeURIComponent(model)}:generateContent?key=${encodeURIComponent(apiKey)}`;
      const prompt = [
        `You are FIRE Copilot — FireSage's Stage 3 Deliberative Quantitative Retirement Strategist (Mode: ${req.mode}).`,
        `GROUNDING RULE: Cite ONLY exact figures from the [EXECUTED GO QUANTITATIVE TOOL EVIDENCE PACK] below. Format concisely in Markdown (headings, bullets, compact tables) and append 'FOLLOW_UPS: Q1? | Q2? | Q3?' on the final line.`,
        '',
        `[EXECUTED GO QUANTITATIVE TOOL EVIDENCE PACK]`,
        wasmResult.evidence_pack,
        '',
        `[USER QUESTION]: ${req.question}`,
      ].join('\n');
      const llmResp = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          contents: [{ role: 'user', parts: [{ text: prompt }] }],
          generationConfig: { temperature: 0.25, maxOutputTokens: 2048 },
        }),
      });
      if (llmResp.ok) {
        const data = await llmResp.json();
        const parts: Array<{ text?: string; thought?: boolean }> =
          data?.candidates?.[0]?.content?.parts || [];
        let rawText = parts
          .filter((p) => !p.thought && p.text)
          .map((p) => p.text)
          .join('')
          .trim();
        if (rawText) {
          const fuMatch = rawText.match(/^FOLLOW_UPS:\s*(.+)$/im);
          if (fuMatch) {
            const followUps = fuMatch[1]
              .split('|')
              .map((s) => s.trim())
              .filter(Boolean)
              .slice(0, 3);
            rawText = rawText.replace(/^FOLLOW_UPS:\s*(.+)$/im, '').trim();
            const uiActions = (wasmResult.actions || []).filter((a) => a.ui_command);
            wasmResult.actions = [
              ...uiActions,
              ...followUps.map((q) => ({ label: `💬 ${q}`, prompt: q })),
            ].slice(0, 5);
          }
          wasmResult.markdown = rawText;
          wasmResult.used_llm = true;
          wasmResult.engine_label = `🧠 ${model} (Gemini API · WASM Planner + ${wasmResult.tool_trace.length} Tools)`;
        }
      }
    } catch {
      // Gracefully keep WASM deterministic synthesis
    }
  }

  return { result: wasmResult, backend: 'go-local-wasm' };
}
