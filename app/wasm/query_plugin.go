package wasm

import (
	"encoding/json"
	wasm "github.com/osmosis-labs/osmosis/v7/app/wasm/bindings"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/types"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type ViewKeeper interface {
	GetPoolState(ctx sdk.Context, poolId uint64) (*types.PoolState, error)
}

func CustomQuerier(osmoKeeper ViewKeeper) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var contractQuery wasm.OsmosisQuery
		if err := json.Unmarshal(request, &contractQuery); err != nil {
			return nil, sdkerrors.Wrap(err, "osmosis query")
		}

		if contractQuery.PoolState != nil {
			poolId := contractQuery.PoolState.PoolId

			state, err := osmoKeeper.GetPoolState(ctx, poolId)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "osmo pool state query")
			}

			var assets []wasmvmtypes.Coin
			for _, asset := range state.Assets {
				assets = append(assets, wasmvmtypes.NewCoin(asset.Amount.Uint64(), asset.Denom))

			}

			res := wasm.PoolStateResponse{
				Assets: assets,
				Shares: wasmvmtypes.NewCoin(state.Shares.Amount.Uint64(), state.Shares.Denom),
			}
			bz, err := json.Marshal(res)
			if err != nil {
				return nil, sdkerrors.Wrap(err, "osmo pool state query response")
			}
			return bz, nil
		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown osmosis query variant"}
	}
}
