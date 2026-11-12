import type { FullAnalysisResponse, MetadataResponse, PlanInput } from '../types';

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

function callWasmWorker<T>(type: 'metadata' | 'analyze', payload?: string): Promise<T> {
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
