package wasm

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	wasmbindings "github.com/osmosis-labs/osmosis/v7/app/wasm/bindings"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/types"
	gammkeeper "github.com/osmosis-labs/osmosis/v7/x/gamm/keeper"
)

type QueryPlugin struct {
	gammKeeper *gammkeeper.Keeper
}

// NewQueryPlugin constructor
func NewQueryPlugin(
	gammK *gammkeeper.Keeper,
) *QueryPlugin {
	return &QueryPlugin{
		gammKeeper: gammK,
	}
}

func (qp QueryPlugin) GetPoolState(ctx sdk.Context, poolId uint64) (*types.PoolState, error) {
	poolData, err := qp.gammKeeper.GetPool(ctx, poolId)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "gamm get pool")
	}
	var poolState types.PoolState
	poolAssets := poolData.GetAllPoolAssets()
	for _, asset := range poolAssets {
		poolState.Assets = append(poolState.Assets, asset.Token)
	}
	poolState.Shares = poolData.GetTotalShares()

	return &poolState, nil
}

func (qp QueryPlugin) GetSpotPrice(ctx sdk.Context, spotPrice *wasmbindings.SpotPrice) (*sdk.Dec, error) {
	if spotPrice == nil {
		return nil, wasmvmtypes.InvalidRequest{Err: "gamm spot price null"}
	}
	poolId := spotPrice.Swap.PoolId
	denomIn := spotPrice.Swap.DenomIn
	denomOut := spotPrice.Swap.DenomOut
	withSwapFee := spotPrice.WithSwapFee
	var price sdk.Dec
	var err error
	if withSwapFee {
		price, err = qp.gammKeeper.CalculateSpotPriceWithSwapFee(ctx, poolId, denomIn, denomOut)
	} else {
		price, err = qp.gammKeeper.CalculateSpotPrice(ctx, poolId, denomIn, denomOut)
	}
	if err != nil {
		return nil, sdkerrors.Wrap(err, "gamm get spot price")
	}
	return &price, nil
}

func (qp QueryPlugin) EstimatePrice(ctx sdk.Context, estimatePrice *wasmbindings.EstimatePrice) (*wasmbindings.SwapAmount, error) {
	if estimatePrice == nil {
		return nil, wasmvmtypes.InvalidRequest{Err: "gamm estimate price null"}
	}
	if err := sdk.ValidateDenom(estimatePrice.First.DenomIn); err != nil {
		return nil, sdkerrors.Wrap(err, "gamm estimate price denom in")
	}
	if err := sdk.ValidateDenom(estimatePrice.First.DenomOut); err != nil {
		return nil, sdkerrors.Wrap(err, "gamm estimate price denom out")
	}
	contractAddr, err := sdk.AccAddressFromBech32(estimatePrice.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "gamm estimate price sender address")
	}

	if estimatePrice.Amount == (wasmbindings.SwapAmount{}) {
		return nil, wasmvmtypes.InvalidRequest{Err: "gamm estimate price empty swap"}
	}

	estimate, err := PerformSwap(qp.gammKeeper, ctx, contractAddr, estimatePrice.ToSwapMsg())
	return estimate, err
}
