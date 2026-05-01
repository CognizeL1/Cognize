package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/state/types"
)

// reputationBonusPercent is kept for legacy reward logic and tests.
func reputationBonusPercent(reputation uint64) int64 {
	switch {
	case reputation >= 90:
		return 20
	case reputation >= 70:
		return 15
	case reputation >= 50:
		return 10
	case reputation >= 30:
		return 5
	default:
		return 0
	}
}

// DistributeEpochRewards distributes contribution rewards to online states
// proportional to their effective weight: Stake × (1 + ReputationBonus% + AIBonus%).
func (k Keeper) DistributeEpochRewards(ctx sdk.Context, epoch uint64) {
	pool := k.getRewardPool(ctx)
	if pool.IsZero() {
		return
	}

	type weightedState struct {
		address string
		weight  *big.Int
	}

	var states []weightedState
	totalWeight := new(big.Int)

	k.IterateStates(ctx, func(state types.State) bool {
		if state.Status != types.StateStatus_STATE_STATUS_ONLINE {
			return false
		}

		stakeAmount := state.StakeAmount.Amount.BigInt()

		repBonus := reputationBonusPercent(state.Reputation)
		aiBonus := k.GetAIBonus(ctx, state.Address)
		multiplier := int64(100) + repBonus + aiBonus
		if multiplier < 10 {
			multiplier = 10
		}

		w := new(big.Int).Mul(stakeAmount, big.NewInt(multiplier))
		totalWeight.Add(totalWeight, w)

		states = append(states, weightedState{
			address: state.Address,
			weight:  w,
		})

		return false
	})

	if totalWeight.Sign() <= 0 || len(states) == 0 {
		return
	}

	poolBig := pool.Amount.BigInt()
	distributed := sdk.NewInt64Coin("aaxon", 0)

	for _, wa := range states {
		share := new(big.Int).Mul(poolBig, wa.weight)
		share.Div(share, totalWeight)

		reward := sdk.NewCoin("aaxon", sdkmath.NewIntFromBigInt(share))
		if reward.IsZero() {
			continue
		}

		recipientAddr, err := sdk.AccAddressFromBech32(wa.address)
		if err != nil {
			continue
		}

		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, sdk.NewCoins(reward)); err != nil {
			k.Logger(ctx).Error("failed to distribute reward", "address", wa.address, "error", err)
			continue
		}

		distributed = distributed.Add(reward)
	}

	var remaining sdk.Coin
	if distributed.Amount.GT(pool.Amount) {
		k.Logger(ctx).Error("distributed exceeds pool — clamping to zero",
			"pool", pool, "distributed", distributed)
		remaining = sdk.NewInt64Coin("aaxon", 0)
	} else {
		remaining = pool.Sub(distributed)
	}
	k.setRewardPool(ctx, remaining)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"epoch_rewards_distributed",
		sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
		sdk.NewAttribute("total_distributed", distributed.String()),
		sdk.NewAttribute("remaining_pool", remaining.String()),
		sdk.NewAttribute("states_count", fmt.Sprintf("%d", len(states))),
	))
}

func (k Keeper) getRewardPool(ctx sdk.Context) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.RewardPoolKey))
	if bz == nil {
		return sdk.NewInt64Coin("aaxon", 0)
	}
	var coin sdk.Coin
	k.cdc.MustUnmarshal(bz, &coin)
	return coin
}

func (k Keeper) setRewardPool(ctx sdk.Context, amount sdk.Coin) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&amount)
	store.Set([]byte(types.RewardPoolKey), bz)
}

// AddToRewardPool adds tokens to the reward pool (e.g., from fees or minting).
func (k Keeper) AddToRewardPool(ctx sdk.Context, amount sdk.Coin) {
	current := k.getRewardPool(ctx)
	k.setRewardPool(ctx, current.Add(amount))
}

func (k Keeper) GetRewardPool(ctx sdk.Context) sdk.Coin {
	return k.getRewardPool(ctx)
}

func (k Keeper) SetRewardPool(ctx sdk.Context, coin sdk.Coin) {
	k.setRewardPool(ctx, coin)
}
