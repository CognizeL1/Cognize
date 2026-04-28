package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	IBCMinFeeBps = 10
	IBCMaxTransfer = 1000000000000
)

func (k Keeper) CreateIBCChannel(ctx sdk.Context, counterChainID, portID string, feeBps uint64, minAmount, maxAmount string) (string, error) {
	if feeBps < IBCMinFeeBps {
		return "", types.ErrIBCFeeTooLow
	}

	channelID := fmt.Sprintf("channel-%s-%d", counterChainID[:8], ctx.BlockHeight())

	channel := types.IBCChannel{
		ChannelID:     channelID,
		PortID:       portID,
		CounterChainID: counterChainID,
		State:        "open",
		FeeBps:       feeBps,
		MinAmount:    minAmount,
		MaxAmount:   maxAmount,
		Enabled:    true,
		CreatedAt:  ctx.BlockHeight(),
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&channel)
	store.Set(types.KeyIBCChannel(counterChainID), bz)

	return channelID, nil
}

func (k Keeper) InitiateIBCTransfer(ctx sdk.Context, sender, receiver, amount, denom, targetChain string) (string, error) {
	amountInt, _ := math.NewIntFromString(amount)
	if amountInt.IsZero() || amountInt.GT(math.NewInt(IBCMaxTransfer)) {
		return "", types.ErrIBCFeeTooLow
	}

	_, found := k.GetAgent(ctx, sender)
	if !found {
		return "", types.ErrAgentNotFound
	}

	channelKey := types.KeyIBCChannel(targetChain)
	channelBz := ctx.KVStore(k.storeKey).Get(channelKey)
	if channelBz == nil {
		return "", types.ErrIBCChannelNotFound
	}

	var channel types.IBCChannel
	json.Unmarshal(channelBz, &channel)

	if !channel.Enabled {
		return "", types.ErrIBCChannelNotFound
	}

	fee := amountInt.Mul(math.NewInt(int64(channel.FeeBps))).Quo(math.NewInt(10000))
	transferAmount := amountInt.Sub(fee)

	transferID := fmt.Sprintf("ibc-%d-%s", ctx.BlockHeight(), sender[:8])

	transfer := types.IBCTransfer{
		TransferID:   transferID,
		Sender:     sender,
		Receiver:  receiver,
		Amount:    transferAmount.String(),
		Denom:     denom,
		Fee:       fee.String(),
		Status:    "pending",
		SourceChain: "cognize",
		TargetChain: targetChain,
		CreatedAt:  ctx.BlockHeight(),
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&transfer)
	store.Set(types.KeyIBCTransfer(transferID), bz)

	senderAddr := sdk.MustAccAddressFromBech32(sender)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, sdk.NewCoins(sdk.NewCoin(denom, amountInt))); err != nil {
		return "", err
	}

	return transferID, nil
}

func (k Keeper) CompleteIBCTransfer(ctx sdk.Context, transferID string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyIBCTransfer(transferID))
	if bz == nil {
		return types.ErrIBCChannelNotFound
	}

	var transfer types.IBCTransfer
	json.Unmarshal(bz, &transfer)

	if transfer.Status != "pending" {
		return types.ErrIBCChannelOpen
	}

	transfer.Status = "completed"
	transfer.CompletedAt = ctx.BlockHeight()
	bz, _ = json.Marshal(&transfer)
	store.Set(types.KeyIBCTransfer(transferID), bz)

	return nil
}

func (k Keeper) CreateMultisigWallet(ctx sdk.Context, creator string, name string, owners []string, threshold uint64) (string, error) {
	if len(owners) < 2 {
		return "", types.ErrMultiSigNotFound
	}
	if threshold > uint64(len(owners)) || threshold == 0 {
		return "", types.ErrMultiSigInsufficient
	}

	walletID := fmt.Sprintf("multisig-%d-%s", ctx.BlockHeight(), creator[:8])

	wallet := types.MultisigWallet{
		WalletID:   walletID,
		Name:     name,
		Owners:   owners,
		Threshold: threshold,
		CreatedBy: creator,
		Balance:  "0",
		CreatedAt: ctx.BlockHeight(),
		Status:   "active",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&wallet)
	store.Set(types.KeyMultisigWallet(walletID), bz)

	return walletID, nil
}

func (k Keeper) ProposeMultisigTx(ctx sdk.Context, walletID, proposer, to, amount, memo string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyMultisigWallet(walletID))
	if bz == nil {
		return "", types.ErrMultiSigNotFound
	}

	var wallet types.MultisigWallet
	json.Unmarshal(bz, &wallet)

	isOwner := false
	for _, owner := range wallet.Owners {
		if owner == proposer {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return "", types.ErrMultiSigNotAuthorized
	}

	txID := fmt.Sprintf("mtx-%d-%s", ctx.BlockHeight(), proposer[:8])

	tx := types.MultisigTx{
		TxID:       txID,
		WalletID:   walletID,
		ProposedBy: proposer,
		To:        to,
		Amount:    amount,
		Memo:     memo,
		Signatures: []string{proposer},
		Executed:  false,
		CreatedAt: ctx.BlockHeight(),
	}

	bz, _ = json.Marshal(&tx)
	store.Set(types.KeyMultisigTx(txID), bz)

	return txID, nil
}

func (k Keeper) SignMultisigTx(ctx sdk.Context, txID, signer string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyMultisigTx(txID))
	if bz == nil {
		return types.ErrMultiSigNotFound
	}

	var tx types.MultisigTx
	json.Unmarshal(bz, &tx)

	if tx.Executed {
		return types.ErrMultiSigInsufficient
	}

	walletBz := store.Get(types.KeyMultisigWallet(tx.WalletID))
	if walletBz == nil {
		return types.ErrMultiSigNotFound
	}

	var wallet types.MultisigWallet
	json.Unmarshal(walletBz, &wallet)

	isOwner := false
	for _, owner := range wallet.Owners {
		if owner == signer {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return types.ErrMultiSigNotAuthorized
	}

	for _, sig := range tx.Signatures {
		if sig == signer {
			return types.ErrAlreadyVoted
		}
	}

	tx.Signatures = append(tx.Signatures, signer)

	if uint64(len(tx.Signatures)) >= wallet.Threshold {
		amount, _ := math.NewIntFromString(tx.Amount)
		recipientAddr := sdk.MustAccAddressFromBech32(tx.To)

		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("acognize", amount))); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, sdk.NewCoins(sdk.NewCoin("acognize", amount))); err != nil {
			return err
		}

		tx.Executed = true
		tx.ExecutedAt = ctx.BlockHeight()
	}

	bz, _ = json.Marshal(&tx)
	store.Set(types.KeyMultisigTx(txID), bz)

	return nil
}

func (k Keeper) CreateDAO(ctx sdk.Context, creator, name, token string, members []string, quorum, threshold uint64) (string, error) {
	daoID := fmt.Sprintf("dao-%d-%s", ctx.BlockHeight(), creator[:8])

	dao := types.DAO{
		DAOID:     daoID,
		Name:      name,
		Creator:   creator,
		Members:   members,
		Token:    token,
		Quorum:   quorum,
		Threshold: threshold,
		Treasurer: creator,
		CreatedAt: ctx.BlockHeight(),
		Status:   "active",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&dao)
	store.Set(types.KeyDAO(daoID), bz)

	return daoID, nil
}

func (k Keeper) JoinDAO(ctx sdk.Context, daoID, member string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyDAO(daoID))
	if bz == nil {
		return types.ErrDAONotFound
	}

	var dao types.DAO
	json.Unmarshal(bz, &dao)

	for _, m := range dao.Members {
		if m == member {
			return types.ErrDAOAlreadyMember
		}
	}

	dao.Members = append(dao.Members, member)
	bz, _ = json.Marshal(&dao)
	store.Set(types.KeyDAO(daoID), bz)

	return nil
}

func (k Keeper) CreateDAOProposal(ctx sdk.Context, daoID, creator, title, action, amount string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyDAO(daoID))
	if bz == nil {
		return "", types.ErrDAONotFound
	}

	proposalID := fmt.Sprintf("prop-%d-%s", ctx.BlockHeight(), creator[:8])

	proposal := types.DAOProposal{
		ProposalID: proposalID,
		DAOID:    daoID,
		Title:    title,
		Action:   action,
		Amount:  amount,
		Votes:   0,
		Status:  "pending",
		CreatedAt: ctx.BlockHeight(),
	}

	daoBz := store.Get(types.KeyDAO(daoID))
	var dao types.DAO
	json.Unmarshal(daoBz, &dao)
	dao.Proposals = append(dao.Proposals, proposal)

	daoBz, _ = json.Marshal(&dao)
	store.Set(types.KeyDAO(daoID), daoBz)

	return proposalID, nil
}

func (k Keeper) CreateStreamingPlan(ctx sdk.Context, sender, recipient, totalAmount, perBlock string, durationBlocks int64) (string, error) {
	senderAgent, found := k.GetAgent(ctx, sender)
	if !found {
		return "", types.ErrAgentNotFound
	}

	total, _ := math.NewIntFromString(totalAmount)
	if senderAgent.StakeAmount.Amount.LT(total) {
		return "", types.ErrInsufficientStake
	}

	planID := fmt.Sprintf("stream-%d-%s", ctx.BlockHeight(), sender[:8])
	startBlock := ctx.BlockHeight()
	endBlock := startBlock + durationBlocks

	plan := types.StreamingPlan{
		PlanID:          planID,
		Sender:         sender,
		Recipient:      recipient,
		TotalAmount:    totalAmount,
		PerBlock:     perBlock,
		BlocksRemaining: uint64(durationBlocks),
		StartBlock:   startBlock,
		EndBlock:    endBlock,
		Active:     true,
		Paused:     false,
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&plan)
	store.Set(types.KeyStreamingPlan(planID), bz)

	senderAddr := sdk.MustAccAddressFromBech32(sender)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, sdk.NewCoins(sdk.NewCoin("acognize", total))); err != nil {
		return "", err
	}

	return planID, nil
}

func (k Keeper) ProcessStreamingPayments(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)

	iter := store.Iterator([]byte("streaming/plan/"), []byte("streaming/plan0"))
	for iter.Valid() {
		var plan types.StreamingPlan
		json.Unmarshal(iter.Value(), &plan)

		if !plan.Active || plan.Paused {
			iter.Next()
			continue
		}

		if ctx.BlockHeight() < plan.StartBlock || ctx.BlockHeight() > plan.EndBlock {
			iter.Next()
			continue
		}

		perBlockAmt, _ := math.NewIntFromString(plan.PerBlock)
		if perBlockAmt.IsZero() {
			iter.Next()
			continue
		}

		recipientAddr := sdk.MustAccAddressFromBech32(plan.Recipient)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, sdk.NewCoins(sdk.NewCoin("acognize", perBlockAmt))); err != nil {
			iter.Next()
			continue
		}

		plan.BlocksRemaining--

		if plan.BlocksRemaining == 0 {
			plan.Active = false
		}

		bz, _ := json.Marshal(&plan)
		store.Set(iter.Key(), bz)

		iter.Next()
	}
	iter.Close()
}