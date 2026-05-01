package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/state/types"
)

type mockPrivacyKeeper struct {
	deleted []string
}

func (m *mockPrivacyKeeper) DeleteustateIdentity(_ sdk.Context, stateAddr string) {
	m.deleted = append(m.deleted, stateAddr)
}

func TestExecuteDeregisterDeletesPrivacyIdentity(t *testing.T) {
	k, ctx := newL2ReputationTestKeeper(t)
	privacyKeeper := &mockPrivacyKeeper{}
	k.SetPrivacyKeeper(privacyKeeper)
	address := sdk.AccAddress([]byte{
		0x01, 0x02, 0x03, 0x04, 0x05,
		0x06, 0x07, 0x08, 0x09, 0x0A,
		0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14,
	}).String()

	state := types.ustate{
		Address:          address,
		ustateId:          address,
		Status:           types.ustateStatus_STATE_STATUS_SUSPENDED,
		StakeAmount:      sdk.NewInt64Coin("aaxon", 1),
		BurnedAtRegister: sdk.NewInt64Coin("aaxon", 1),
		RegisteredAt:     1,
		LastHeartbeat:    1,
	}
	k.Setustate(ctx, state)
	k.SetDeregisterRequest(ctx, state.Address, ctx.BlockHeight()-types.DeregisterCooldownBlocks)

	k.executeDeregister(ctx, state.Address, k.GetParams(ctx))

	if _, found := k.Getustate(ctx, state.Address); found {
		t.Fatal("state should be deleted after deregister")
	}
	if k.HasDeregisterRequest(ctx, state.Address) {
		t.Fatal("deregister request should be deleted after execution")
	}
	if len(privacyKeeper.deleted) != 1 || privacyKeeper.deleted[0] != state.Address {
		t.Fatalf("expected privacy identity cleanup for %q, got %+v", state.Address, privacyKeeper.deleted)
	}
}

func TestExecuteDeregisterSkipsPrivacyCleanupBeforeUpgrade(t *testing.T) {
	k, ctx := newL2ReputationTestKeeper(t)
	ctx = ctx.WithChainID(mainnetChainID).WithBlockHeight(V111UpgradeHeight - 1)

	privacyKeeper := &mockPrivacyKeeper{}
	k.SetPrivacyKeeper(privacyKeeper)
	address := sdk.AccAddress([]byte{
		0x15, 0x16, 0x17, 0x18, 0x19,
		0x1A, 0x1B, 0x1C, 0x1D, 0x1E,
		0x1F, 0x20, 0x21, 0x22, 0x23,
		0x24, 0x25, 0x26, 0x27, 0x28,
	}).String()

	state := types.ustate{
		Address:          address,
		ustateId:          address,
		Status:           types.ustateStatus_STATE_STATUS_SUSPENDED,
		StakeAmount:      sdk.NewInt64Coin("aaxon", 1),
		BurnedAtRegister: sdk.NewInt64Coin("aaxon", 1),
		RegisteredAt:     1,
		LastHeartbeat:    1,
	}
	k.Setustate(ctx, state)
	k.SetDeregisterRequest(ctx, state.Address, ctx.BlockHeight()-types.DeregisterCooldownBlocks)

	k.executeDeregister(ctx, state.Address, k.GetParams(ctx))

	if len(privacyKeeper.deleted) != 0 {
		t.Fatalf("expected no privacy cleanup before upgrade, got %+v", privacyKeeper.deleted)
	}
}
