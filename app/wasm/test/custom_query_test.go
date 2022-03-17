package wasm

import (
	"encoding/json"
	"fmt"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/osmosis-labs/osmosis/v7/app/wasm"
	"github.com/osmosis-labs/osmosis/v7/x/gamm/pool-models/balancer"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/osmosis-labs/osmosis/v7/app"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/bindings"
	gammtypes "github.com/osmosis-labs/osmosis/v7/x/gamm/types"
)

// we must pay this many uosmo for every pool we create
var poolFee int64 = 1000000000

var defaultFunds = sdk.NewCoins(
	sdk.NewInt64Coin("uatom", 333000000),
	sdk.NewInt64Coin("uosmo", 555000000+2*poolFee),
	sdk.NewInt64Coin("ustar", 999000000),
)

func SetupCustomApp(t *testing.T, addr sdk.AccAddress) (*app.OsmosisApp, sdk.Context) {
	osmosis, ctx := CreateTestInput()
	wasmKeeper := osmosis.WasmKeeper

	storeReflectCode(t, ctx, osmosis, addr)

	cInfo := wasmKeeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, cInfo)

	return osmosis, ctx
}

func TestQueryFullDenom(t *testing.T) {
	actor := RandomAccountAddress()
	osmosis, ctx := SetupCustomApp(t, actor)

	reflect := instantiateReflectContract(t, ctx, osmosis, actor)
	require.NotEmpty(t, reflect)

	// query full denom
	query := wasmbindings.OsmosisQuery{
		FullDenom: &wasmbindings.FullDenom{
			Contract: reflect.String(),
			SubDenom: "ustart",
		},
	}
	resp := wasmbindings.FullDenomResponse{}
	queryCustom(t, ctx, osmosis, reflect, query, &resp)

	expected := fmt.Sprintf("cw/%s/ustart", reflect.String())
	require.EqualValues(t, expected, resp.Denom)
}

func TestQueryPool(t *testing.T) {
	actor := RandomAccountAddress()
	osmosis, ctx := SetupCustomApp(t, actor)

	fundAccount(t, ctx, osmosis, actor, defaultFunds)

	poolFunds := []sdk.Coin{
		sdk.NewInt64Coin("uosmo", 12000000),
		sdk.NewInt64Coin("ustar", 240000000),
	}
	// 20 star to 1 osmo
	starPool := preparePool(t, ctx, osmosis, actor, poolFunds)

	pool2Funds := []sdk.Coin{
		sdk.NewInt64Coin("uatom", 6000000),
		sdk.NewInt64Coin("uosmo", 12000000),
	}
	// 20 star to 1 osmo
	atomPool := preparePool(t, ctx, osmosis, actor, pool2Funds)

	reflect := instantiateReflectContract(t, ctx, osmosis, actor)
	require.NotEmpty(t, reflect)

	// query pool state
	query := wasmbindings.OsmosisQuery{
		PoolState: &wasmbindings.PoolState{PoolId: starPool},
	}
	resp := wasmbindings.PoolStateResponse{}
	queryCustom(t, ctx, osmosis, reflect, query, &resp)
	expected := wasm.ConvertSdkCoinsToWasmCoins(poolFunds)
	require.EqualValues(t, expected, resp.Assets)
	assertValidShares(t, resp.Shares, starPool)

	// query second pool state
	query = wasmbindings.OsmosisQuery{
		PoolState: &wasmbindings.PoolState{PoolId: atomPool},
	}
	resp = wasmbindings.PoolStateResponse{}
	queryCustom(t, ctx, osmosis, reflect, query, &resp)
	expected = wasm.ConvertSdkCoinsToWasmCoins(pool2Funds)
	require.EqualValues(t, expected, resp.Assets)
	assertValidShares(t, resp.Shares, atomPool)
}

func TestQuerySpotPrice(t *testing.T) {
	actor := RandomAccountAddress()
	osmosis, ctx := SetupCustomApp(t, actor)
	swapFee := 0. // FIXME: Set / support an actual fee
	epsilon := 1e-6

	fundAccount(t, ctx, osmosis, actor, defaultFunds)

	poolFunds := []sdk.Coin{
		sdk.NewInt64Coin("uosmo", 12000000),
		sdk.NewInt64Coin("ustar", 240000000),
	}
	// 20 star to 1 osmo
	starPool := preparePool(t, ctx, osmosis, actor, poolFunds)

	reflect := instantiateReflectContract(t, ctx, osmosis, actor)
	require.NotEmpty(t, reflect)

	// query spot price
	query := wasmbindings.OsmosisQuery{
		SpotPrice: &wasmbindings.SpotPrice{
			Swap: wasmbindings.Swap{
				PoolId:   starPool,
				DenomIn:  "ustar",
				DenomOut: "uosmo",
			},
			WithSwapFee: false,
		},
	}
	resp := wasmbindings.SpotPriceResponse{}
	queryCustom(t, ctx, osmosis, reflect, query, &resp)

	price, err := strconv.ParseFloat(resp.Price, 64)
	require.NoError(t, err)

	uosmo, err := poolFunds[0].Amount.ToDec().Float64()
	require.NoError(t, err)
	ustar, err := poolFunds[1].Amount.ToDec().Float64()
	require.NoError(t, err)

	expected := ustar / uosmo
	require.InEpsilonf(t, expected, price, epsilon, fmt.Sprintf("Outside of tolerance (%f)", epsilon))

	// and the reverse conversion (with swap fee)
	// query spot price
	query = wasmbindings.OsmosisQuery{
		SpotPrice: &wasmbindings.SpotPrice{
			Swap: wasmbindings.Swap{
				PoolId:   starPool,
				DenomIn:  "uosmo",
				DenomOut: "ustar",
			},
			WithSwapFee: true,
		},
	}
	resp = wasmbindings.SpotPriceResponse{}
	queryCustom(t, ctx, osmosis, reflect, query, &resp)

	price, err = strconv.ParseFloat(resp.Price, 32)
	require.NoError(t, err)

	expected = 1. / expected
	require.InEpsilonf(t, expected+swapFee, price, epsilon, fmt.Sprintf("Outside of tolerance (%f)", epsilon))
}

type ReflectQuery struct {
	Chain *ChainRequest `json:"chain,omitempty"`
}

type ChainRequest struct {
	Request wasmvmtypes.QueryRequest `json:"request"`
}

type ChainResponse struct {
	Data []byte `json:"data"`
}

func queryCustom(t *testing.T, ctx sdk.Context, osmosis *app.OsmosisApp, contract sdk.AccAddress, request wasmbindings.OsmosisQuery, response interface{}) {
	msgBz, err := json.Marshal(request)
	require.NoError(t, err)

	query := ReflectQuery{
		Chain: &ChainRequest{
			Request: wasmvmtypes.QueryRequest{Custom: msgBz},
		},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)

	resBz, err := osmosis.WasmKeeper.QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var resp ChainResponse
	err = json.Unmarshal(resBz, &resp)
	require.NoError(t, err)
	err = json.Unmarshal(resp.Data, response)
	require.NoError(t, err)
}

func assertValidShares(t *testing.T, shares wasmvmtypes.Coin, poolID uint64) {
	// sanity check: check the denom and ensure at least 18 decimal places
	denom := fmt.Sprintf("gamm/pool/%d", poolID)
	require.Equal(t, denom, shares.Denom)
	require.Greater(t, len(shares.Amount), 18)
}

func storeReflectCode(t *testing.T, ctx sdk.Context, osmosis *app.OsmosisApp, addr sdk.AccAddress) {
	govKeeper := osmosis.GovKeeper
	wasmCode, err := ioutil.ReadFile("../testdata/osmo_reflect.wasm")
	require.NoError(t, err)

	src := wasmtypes.StoreCodeProposalFixture(func(p *wasmtypes.StoreCodeProposal) {
		p.RunAs = addr.String()
		p.WASMByteCode = wasmCode
	})

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, src)
	require.NoError(t, err)

	// and proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx, storedProposal.GetContent())
	require.NoError(t, err)
}

func instantiateReflectContract(t *testing.T, ctx sdk.Context, osmosis *app.OsmosisApp, funder sdk.AccAddress) sdk.AccAddress {
	initMsgBz := []byte("{}")
	contractKeeper := keeper.NewDefaultPermissionKeeper(osmosis.WasmKeeper)
	codeID := uint64(1)
	addr, _, err := contractKeeper.Instantiate(ctx, codeID, funder, funder, initMsgBz, "demo contract", nil)
	require.NoError(t, err)

	return addr
}

func fundAccount(t *testing.T, ctx sdk.Context, osmosis *app.OsmosisApp, addr sdk.AccAddress, coins sdk.Coins) {
	err := simapp.FundAccount(
		osmosis.BankKeeper,
		ctx,
		addr,
		coins,
	)
	require.NoError(t, err)
}

func preparePool(t *testing.T, ctx sdk.Context, osmosis *app.OsmosisApp, addr sdk.AccAddress, funds []sdk.Coin) uint64 {
	var assets []gammtypes.PoolAsset
	for _, coin := range funds {
		assets = append(assets, gammtypes.PoolAsset{
			Weight: sdk.NewInt(100),
			Token:  coin,
		})
	}

	poolParams := balancer.PoolParams{
		SwapFee: sdk.NewDec(0),
		ExitFee: sdk.NewDec(0),
	}

	poolId, err := osmosis.GAMMKeeper.CreateBalancerPool(ctx, addr, poolParams, assets, "")
	require.NoError(t, err)
	return poolId
}
