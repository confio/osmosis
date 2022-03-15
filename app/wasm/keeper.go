package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/osmosis-labs/osmosis/v7/app/wasm/types"
	gammkeeper "github.com/osmosis-labs/osmosis/v7/x/gamm/keeper"
)

type Keeper struct {
	//codec       codec.Codec
	//storeKey    sdk.StoreKey
	//paramStore  paramtypes.Subspace
	gammKeeper *gammkeeper.Keeper
}

// NewKeeper constructor
func NewKeeper(
	gammK *gammkeeper.Keeper,
) *Keeper {
	// set KeyTable if it has not already been set
	//if !paramSpace.HasKeyTable() {
	//	paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	//}
	// ensure bonded and not bonded module accounts are set
	//if addr := ak.GetModuleAddress(types.BondedPoolName); addr == nil {
	//	panic(fmt.Sprintf("%s module account has not been set", types.BondedPoolName))
	//}

	return &Keeper{
		//codec:       marshaler,
		//storeKey:    key,
		//paramStore:  paramSpace,
		gammKeeper: gammK,
	}
}

func (k Keeper) GetPoolState(ctx sdk.Context, poolId uint64) (*types.PoolState, error) {
	poolData, err := k.gammKeeper.GetPool(ctx, poolId)
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
