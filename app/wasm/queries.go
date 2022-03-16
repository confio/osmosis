package wasm

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	wasm "github.com/osmosis-labs/osmosis/v7/app/wasm/bindings"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/types"
	gammkeeper "github.com/osmosis-labs/osmosis/v7/x/gamm/keeper"
	gammtypes "github.com/osmosis-labs/osmosis/v7/x/gamm/types"
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

func (qp QueryPlugin) GetSpotPrice(ctx sdk.Context, poolId uint64, denomIn string, denomOut string, withSwapFee bool) (*sdk.Dec, error) {
	var spotPrice sdk.Dec
	var err error
	if withSwapFee {
		spotPrice, err = qp.gammKeeper.CalculateSpotPriceWithSwapFee(ctx, poolId, denomIn, denomOut)
	} else {
		spotPrice, err = qp.gammKeeper.CalculateSpotPrice(ctx, poolId, denomIn, denomOut)
	}
	if err != nil {
		return nil, sdkerrors.Wrap(err, "gamm get spot price")
	}
	return &spotPrice, nil
}

func (qp QueryPlugin) EstimatePrice(ctx sdk.Context, sender string, firstPoolId uint64, denomIn string, denomOut string, exactIn bool, amount sdk.Int, route []wasm.Step) (*sdk.Int, error) {
	var senderAddress = sdk.AccAddress(sender)
	var swapAmount sdk.Int
	var err error
	if exactIn {
		tokenIn := sdk.NewCoin(denomIn, amount)
		// Populate route
		var steps gammtypes.SwapAmountInRoutes
		firstStep := gammtypes.SwapAmountInRoute{
			PoolId:        firstPoolId,
			TokenOutDenom: denomOut,
		}
		steps = append(steps, firstStep)
		for _, step := range route {
			step := gammtypes.SwapAmountInRoute{
				PoolId:        step.PoolId,
				TokenOutDenom: step.DenomOut,
			}
			steps = append(steps, step)
		}
		tokenOutMinAmount := sdk.NewInt(1)
		swapAmount, err = qp.gammKeeper.MultihopSwapExactAmountIn(ctx, senderAddress, steps, tokenIn, tokenOutMinAmount)
	} else {
		tokenOut := sdk.NewCoin(denomOut, amount)
		// Populate route
		var steps gammtypes.SwapAmountOutRoutes
		firstStep := gammtypes.SwapAmountOutRoute{
			PoolId:       firstPoolId,
			TokenInDenom: denomIn,
		}
		steps = append(steps, firstStep)
		for _, step := range route {
			step := gammtypes.SwapAmountOutRoute{
				PoolId:       step.PoolId,
				TokenInDenom: step.DenomOut,
			}
			steps = append(steps, step)
		}
		tokenInMaxAmount := sdk.NewInt(math.MaxInt64)
		swapAmount, err = qp.gammKeeper.MultihopSwapExactAmountOut(ctx, senderAddress, steps, tokenInMaxAmount, tokenOut)
	}
	if err != nil {
		return nil, sdkerrors.Wrap(err, "gamm estimate price")
	}
	return &swapAmount, nil
}
