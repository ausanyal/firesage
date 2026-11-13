//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/ausanyal/firesage/internal/copilot"
	"github.com/ausanyal/firesage/internal/engine"
)

func main() {
	js.Global().Set("firesageGetMetadata", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		payload := engine.BuildMetadataPayload()
		b, _ := json.Marshal(payload)
		return string(b)
	}))

	js.Global().Set("firesageRunAnalysis", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return `{"error":"missing JSON input"}`
		}
		var req engine.PlanInput
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			errObj, _ := json.Marshal(map[string]string{"error": err.Error()})
			return string(errObj)
		}
		res := engine.RunFullAnalysis(req)
		b, _ := json.Marshal(res)
		return string(b)
	}))

	js.Global().Set("firesageRunChat", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return `{"ok":false,"error":"missing JSON chat request"}`
		}
		var req copilot.ChatRequest
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			errObj, _ := json.Marshal(map[string]any{"ok": false, "error": err.Error()})
			return string(errObj)
		}
		res := copilot.ExecutePipeline(req)
		res.EngineLabel = "⚡ Go WASM Planner + Stochastic Tool Engine"
		b, _ := json.Marshal(res)
		return string(b)
	}))

	// Keep Go WASM runtime alive
	select {}
}
