package wasm

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func SetupWasmHandlers(
	osmoKeeper ViewKeeper,
) []wasmkeeper.Option {
	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(osmoKeeper),
	})

	return []wasm.Option{
		queryPluginOpt,
	}
}
