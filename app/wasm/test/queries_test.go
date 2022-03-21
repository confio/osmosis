package wasm

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		"existent pool id ": {
			poolId: starPool,
			expPoolState: &types.PoolState{
				Assets: poolFunds,
				Shares: sdk.NewCoin(starSharesDenom, starSharedAmount),
			},
		},
		"non-existent pool id ": {
			poolId: starPool + 1,
			expErr: true,
		},
		"zero pool id ": {
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
