/// <reference lib="webworker" />

declare const self: DedicatedWorkerGlobalScope & {
  Go?: new () => {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
  };
  firesageGetMetadata?: () => string;
  firesageRunAnalysis?: (inputJson: string) => string;
  firesageRunChat?: (reqJson: string) => string;
};

let initPromise: Promise<void> | null = null;

async function ensureWasmReady(): Promise<void> {
  if (initPromise) return initPromise;

  initPromise = (async () => {
    const execResp = await fetch('/wasm_exec.js');
    if (!execResp.ok) {
      throw new Error(`Failed to fetch wasm_exec.js: ${execResp.status}`);
    }
    const execScript = await execResp.text();
    // Evaluate Go's runtime glue in worker global scope (works in ES module workers too)
    (0, eval)(execScript);

    if (!self.Go) {
      throw new Error('Go WASM runtime constructor not initialized');
    }

    const go = new self.Go();
    const wasmResp = await fetch('/firesage.wasm');
    if (!wasmResp.ok) {
      throw new Error(`Failed to fetch firesage.wasm: ${wasmResp.status}`);
    }

    let instance: WebAssembly.Instance;
    if ('instantiateStreaming' in WebAssembly) {
      try {
        const res = await WebAssembly.instantiateStreaming(wasmResp, go.importObject);
        instance = res.instance;
      } catch {
        const fallbackResp = await fetch('/firesage.wasm');
        const bytes = await fallbackResp.arrayBuffer();
        const res = await WebAssembly.instantiate(bytes, go.importObject);
        instance = res.instance;
      }
    } else {
      const bytes = await wasmResp.arrayBuffer();
      const res = await WebAssembly.instantiate(bytes, go.importObject);
      instance = res.instance;
    }

    // Run Go main() asynchronously (it registers global functions and blocks on select{})
    void go.run(instance);

    // Wait briefly until global functions are registered
    for (let i = 0; i < 50; i++) {
      if (
        typeof self.firesageGetMetadata === 'function' &&
        typeof self.firesageRunAnalysis === 'function' &&
        typeof self.firesageRunChat === 'function'
      ) {
        return;
      }
      await new Promise((r) => setTimeout(r, 10));
    }
    throw new Error('Timed out waiting for Go WASM exports');
  })();

  return initPromise;
}

self.addEventListener(
  'message',
  async (event: MessageEvent<{ id: number; type: 'metadata' | 'analyze' | 'chat'; payload?: string }>) => {
    const { id, type, payload } = event.data;
    try {
      await ensureWasmReady();
      if (type === 'metadata') {
        const jsonStr = self.firesageGetMetadata!();
        self.postMessage({ id, ok: true, data: JSON.parse(jsonStr) });
      } else if (type === 'analyze') {
        const jsonStr = self.firesageRunAnalysis!(payload || '{}');
        const parsed = JSON.parse(jsonStr);
        if (parsed && parsed.error) {
          self.postMessage({ id, ok: false, error: parsed.error });
        } else {
          self.postMessage({ id, ok: true, data: parsed });
        }
      } else if (type === 'chat') {
        const jsonStr = self.firesageRunChat!(payload || '{}');
        const parsed = JSON.parse(jsonStr);
        if (parsed && parsed.error) {
          self.postMessage({ id, ok: false, error: parsed.error });
        } else {
          self.postMessage({ id, ok: true, data: parsed });
        }
      }
    } catch (err) {
      self.postMessage({
        id,
        ok: false,
        error: err instanceof Error ? err.message : String(err),
      });
    }
  },
);
