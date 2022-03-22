package wasm

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmbindings "github.com/osmosis-labs/osmosis/v7/app/wasm/bindings"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/types"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osmosis-labs/osmosis/v7/app/wasm"
)

func TestFullDenom(t *testing.T) {
	actor := RandomAccountAddress()

	specs := map[string]struct {
		addr         string
		subDenom     string
		expFullDenom string
		expErr       bool
	}{
		"valid address": {
			addr:         actor.String(),
			subDenom:     "sub_denom_1",
			expFullDenom: fmt.Sprintf("cw/%s/sub_denom_1", actor.String()),
		},
		"empty address": {
			addr:     "",
			subDenom: "sub_denom_1",
			expErr:   true,
		},
		"invalid address": {
			addr:     "invalid",
			subDenom: "sub_denom_1",
			expErr:   true,
		},
		"empty sub-denom": {
			addr:     actor.String(),
			subDenom: "",
			expErr:   true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			gotFullDenom, gotErr := wasm.GetFullDenom(spec.addr, spec.subDenom)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expFullDenom, gotFullDenom, "exp %s but got %s", spec.expFullDenom, gotFullDenom)
		})
	}
}

func TestPoolState(t *testing.T) {
	actor := RandomAccountAddress()
	osmosis, ctx := SetupCustomApp(t, actor)

	fundAccount(t, ctx, osmosis, actor, defaultFunds)

	poolFunds := []sdk.Coin{
		sdk.NewInt64Coin("uosmo", 12000000),
		sdk.NewInt64Coin("ustar", 240000000),
	}
	// 20 star to 1 osmo
	starPool := preparePool(t, ctx, osmosis, actor, poolFunds)

	// FIXME: Derive / obtain these values
	starSharesDenom := fmt.Sprintf("gamm/pool/%d", starPool)
	starSharedAmount, _ := sdk.NewIntFromString("100_000_000_000_000_000_000")

	queryPlugin := wasm.NewQueryPlugin(osmosis.GAMMKeeper)

	specs := map[string]struct {
		poolId       uint64
		expPoolState *types.PoolState
		expErr       bool
	}{
		"existent pool id": {
			poolId: starPool,
			expPoolState: &types.PoolState{
				Assets: poolFunds,
				Shares: sdk.NewCoin(starSharesDenom, starSharedAmount),
			},
		},
		"non-existent pool id": {
			poolId: starPool + 1,
			expErr: true,
		},
		"zero pool id": {
			poolId: 0,
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			gotPoolState, gotErr := queryPlugin.GetPoolState(ctx, spec.poolId)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expPoolState, gotPoolState, "exp %s but got %s", spec.expPoolState, gotPoolState)
		})
	}
}

func TestSpotPrice(t *testing.T) {
	actor := RandomAccountAddress()
	swapFee := 0. // FIXME: Set / support an actual fee
	epsilon := 1e-6
	osmosis, ctx := SetupCustomApp(t, actor)

	fundAccount(t, ctx, osmosis, actor, defaultFunds)

	poolFunds := []sdk.Coin{
		sdk.NewInt64Coin("uosmo", 12000000),
		sdk.NewInt64Coin("ustar", 240000000),
	}
	// 20 star to 1 osmo
	starPool := preparePool(t, ctx, osmosis, actor, poolFunds)
	starPrice := sdk.MustNewDecFromStr(fmt.Sprintf("%f", 12000000./240000000))
	starFee := sdk.MustNewDecFromStr(fmt.Sprintf("%f", swapFee))
	starPriceWithFee := starPrice.Add(starFee)

	queryPlugin := wasm.NewQueryPlugin(osmosis.GAMMKeeper)

	specs := map[string]struct {
		spotPrice *wasmbindings.SpotPrice
		expPrice  *sdk.Dec
		expErr    bool
	}{
		"valid spot price": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "uosmo",
					DenomOut: "ustar",
				},
				WithSwapFee: false,
			},
			expPrice: &starPrice,
		},
		"valid spot price with fee": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "uosmo",
					DenomOut: "ustar",
				},
				WithSwapFee: true,
			},
			expPrice: &starPriceWithFee,
		},
		"non-existent pool id": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool + 2,
					DenomIn:  "uosmo",
					DenomOut: "ustar",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"zero pool id": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   0,
					DenomIn:  "uosmo",
					DenomOut: "ustar",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"invalid denom in": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "invalid",
					DenomOut: "ustar",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"empty denom in": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "",
					DenomOut: "ustar",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"invalid denom out": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "uosmo",
					DenomOut: "invalid",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"empty denom out": {
			spotPrice: &wasmbindings.SpotPrice{
				Swap: wasmbindings.Swap{
					PoolId:   starPool,
					DenomIn:  "uosmo",
					DenomOut: "",
				},
				WithSwapFee: false,
			},
			expErr: true,
		},
		"null spot price": {
			spotPrice: nil,
			expErr:    true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			gotPrice, gotErr := queryPlugin.GetSpotPrice(ctx, spec.spotPrice)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.InEpsilonf(t, spec.expPrice.MustFloat64(), gotPrice.MustFloat64(), epsilon, "exp %s but got %s", spec.expPrice.String(), gotPrice.String())
		})
	}
}
