package vote

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lino-network/lino/tx/vote/model"
	"github.com/lino-network/lino/types"
	"github.com/stretchr/testify/assert"
)

var (
	l400  = types.LNO("400")
	l1000 = types.LNO("1000")
	l1600 = types.LNO("1600")
	l2000 = types.LNO("2000")

	c400  = types.Coin{400 * types.Decimals}
	c600  = types.Coin{600 * types.Decimals}
	c1000 = types.Coin{1000 * types.Decimals}
	c1200 = types.Coin{1200 * types.Decimals}
	c1600 = types.Coin{1600 * types.Decimals}
	c2000 = types.Coin{2000 * types.Decimals}
	c3200 = types.Coin{3200 * types.Decimals}
	c3600 = types.Coin{3600 * types.Decimals}
	c4600 = types.Coin{4600 * types.Decimals}
)

func TestVoterDepositBasic(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	handler := NewHandler(vm, am, gm)

	// create two test users
	user1 := createTestAccount(ctx, am, "user1")
	am.AddSavingCoin(ctx, user1, c3600)

	// let user1 register as voter
	msg := NewVoterDepositMsg("user1", l1600)
	result := handler(ctx, msg)
	assert.Equal(t, sdk.Result{}, result)
	handler(ctx, msg)

	// check acc1's money has been withdrawn
	acc1saving, _ := am.GetBankSaving(ctx, user1)
	assert.Equal(t, c400.Plus(initCoin), acc1saving)
	assert.Equal(t, true, vm.IsVoterExist(ctx, user1))

	// make sure the voter's account info is correct
	voter, _ := vm.storage.GetVoter(ctx, user1)
	assert.Equal(t, c3200, voter.Deposit)
}

func TestDelegateBasic(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	handler := NewHandler(vm, am, gm)

	// create test users
	user1 := createTestAccount(ctx, am, "user1")
	am.AddSavingCoin(ctx, user1, c2000)

	user2 := createTestAccount(ctx, am, "user2")
	am.AddSavingCoin(ctx, user2, c2000)

	user3 := createTestAccount(ctx, am, "user3")
	am.AddSavingCoin(ctx, user3, c2000)

	// let user1 register as voter
	msg := NewVoterDepositMsg("user1", l1600)
	handler(ctx, msg)

	// let user2 delegate power to user1 twice
	msg2 := NewDelegateMsg("user2", "user1", l1000)
	handler(ctx, msg2)
	result2 := handler(ctx, msg2)
	assert.Equal(t, sdk.Result{}, result2)

	// make sure the voter's voting power is correct
	voter, _ := vm.storage.GetVoter(ctx, user1)
	assert.Equal(t, c1600, voter.Deposit)
	assert.Equal(t, c2000, voter.DelegatedPower)

	votingPower, _ := vm.GetVotingPower(ctx, "user1")
	assert.Equal(t, true, votingPower.IsEqual(c3600))
	acc2Balance, _ := am.GetBankSaving(ctx, user2)
	assert.Equal(t, acc2Balance, initCoin)

	// let user3 delegate power to user1
	msg3 := NewDelegateMsg("user3", "user1", l1000)
	result3 := handler(ctx, msg3)
	assert.Equal(t, sdk.Result{}, result3)

	// check delegator list is correct
	delegators, _ := vm.storage.GetAllDelegators(ctx, "user1")
	assert.Equal(t, 2, len(delegators))
	assert.Equal(t, user2, delegators[0])
	assert.Equal(t, user3, delegators[1])

	// check delegation are correct
	delegation1, _ := vm.storage.GetDelegation(ctx, "user1", "user2")
	delegation2, _ := vm.storage.GetDelegation(ctx, "user1", "user3")
	assert.Equal(t, c2000, delegation1.Amount)
	assert.Equal(t, c1000, delegation2.Amount)
}

func TestRevokeBasic(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	handler := NewHandler(vm, am, gm)

	// create test users
	user1 := createTestAccount(ctx, am, "user1")
	am.AddSavingCoin(ctx, user1, c2000)

	user2 := createTestAccount(ctx, am, "user2")
	am.AddSavingCoin(ctx, user2, c2000)

	user3 := createTestAccount(ctx, am, "user3")
	am.AddSavingCoin(ctx, user3, c2000)

	// let user1 register as voter
	msg := NewVoterDepositMsg("user1", l1600)
	handler(ctx, msg)

	// let user2 delegate power to user1
	msg2 := NewDelegateMsg("user2", "user1", l1000)
	handler(ctx, msg2)

	// let user3 delegate power to user1
	msg3 := NewDelegateMsg("user3", "user1", l1000)
	handler(ctx, msg3)
	_, res := vm.storage.GetDelegation(ctx, "user1", "user3")
	assert.Nil(t, res)

	// let user3 reovke delegation
	msg4 := NewRevokeDelegationMsg("user3", "user1")
	result := handler(ctx, msg4)
	assert.Equal(t, sdk.Result{}, result)

	// make sure user3 won't get coins immediately, but user1 power down immediately
	voter, _ := vm.storage.GetVoter(ctx, "user1")
	acc3Balance, _ := am.GetBankSaving(ctx, user3)
	_, err := vm.storage.GetDelegation(ctx, "user1", "user3")
	assert.Equal(t, ErrGetDelegation(), err)
	assert.Equal(t, c1000, voter.DelegatedPower)
	assert.Equal(t, acc3Balance, c1000.Plus(initCoin))

	// set user1 as validator (cannot revoke)
	referenceList := &model.ReferenceList{
		AllValidators: []types.AccountKey{user1},
	}
	vm.storage.SetReferenceList(ctx, referenceList)
	msg5 := NewVoterRevokeMsg("user1")
	result2 := handler(ctx, msg5)
	assert.Equal(t, ErrValidatorCannotRevoke().Result(), result2)

	// invalid user cannot revoke
	invalidMsg := NewVoterRevokeMsg("wqwdqwdasdsa")
	resultInvalid := handler(ctx, invalidMsg)
	assert.Equal(t, ErrGetVoter().Result(), resultInvalid)

	//  user1  can revoke voter candidancy now
	referenceList = &model.ReferenceList{
		AllValidators: []types.AccountKey{},
	}
	vm.storage.SetReferenceList(ctx, referenceList)
	result3 := handler(ctx, msg5)
	assert.Equal(t, sdk.Result{}, result3)

	// make sure user2 wont get coins immediately, and delegatin was deleted
	_, err2 := vm.storage.GetVoter(ctx, "user1")
	acc1Balance, _ := am.GetBankSaving(ctx, user1)
	acc2Balance, _ := am.GetBankSaving(ctx, user2)
	assert.Equal(t, ErrGetDelegation(), err)
	assert.Equal(t, ErrGetVoter(), err2)
	assert.Equal(t, c400.Plus(initCoin), acc1Balance)
	assert.Equal(t, c1000.Plus(initCoin), acc2Balance)
}

func TestVoterWithdraw(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	handler := NewHandler(vm, am, gm)

	user1 := createTestAccount(ctx, am, "user1")
	am.AddSavingCoin(ctx, user1, c3600)

	// withdraw will fail if hasn't registed as voter
	illegalWithdrawMsg := NewVoterWithdrawMsg("user1", l1600)
	res := handler(ctx, illegalWithdrawMsg)
	assert.Equal(t, ErrIllegalWithdraw().Result(), res)

	// let user1 register as voter
	msg := NewVoterDepositMsg("user1", l1600)
	handler(ctx, msg)

	msg2 := NewVoterWithdrawMsg("user1", l1000)
	result2 := handler(ctx, msg2)
	assert.Equal(t, ErrIllegalWithdraw().Result(), result2)

	msg3 := NewVoterWithdrawMsg("user1", l400)
	result3 := handler(ctx, msg3)
	assert.Equal(t, sdk.Result{}, result3)

	voter, _ := vm.storage.GetVoter(ctx, "user1")
	assert.Equal(t, c1200, voter.Deposit)
}

func TestVoteBasic(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	handler := NewHandler(vm, am, gm)

	proposalID := int64(1)
	user1 := createTestAccount(ctx, am, "user1")
	am.AddSavingCoin(ctx, user1, c2000)

	user2 := createTestAccount(ctx, am, "user2")
	am.AddSavingCoin(ctx, user2, c2000)

	user3 := createTestAccount(ctx, am, "user3")
	am.AddSavingCoin(ctx, user3, c2000)

	// let user1 create a proposal
	referenceList := &model.ReferenceList{
		OngoingProposal: []types.ProposalKey{types.ProposalKey("1")},
	}
	vm.storage.SetReferenceList(ctx, referenceList)

	// must become a voter before voting
	voteMsg := NewVoteMsg("user2", proposalID, true)
	result2 := handler(ctx, voteMsg)
	assert.Equal(t, ErrGetVoter().Result(), result2)

	depositMsg := NewVoterDepositMsg("user2", l1000)
	depositMsg2 := NewVoterDepositMsg("user3", l2000)
	handler(ctx, depositMsg)
	handler(ctx, depositMsg2)

	// invalid deposit
	invalidDepositMsg := NewVoterDepositMsg("1du1i2bdi12bud", l2000)
	res := handler(ctx, invalidDepositMsg)
	assert.Equal(t, ErrUsernameNotFound().Result(), res)

	// Now user2 can vote, vote on a non exist proposal
	invalidaVoteMsg := NewVoteMsg("user3", 10, true)
	voteRes := handler(ctx, invalidaVoteMsg)
	assert.Equal(t, ErrNotOngoingProposal().Result(), voteRes)

	// successfully vote
	voteMsg2 := NewVoteMsg("user2", proposalID, true)
	voteMsg3 := NewVoteMsg("user3", proposalID, true)
	handler(ctx, voteMsg2)
	handler(ctx, voteMsg3)

	// user cannot vote again
	voteAgainMsg := NewVoteMsg("user3", proposalID, false)
	res = handler(ctx, voteAgainMsg)
	assert.Equal(t, ErrVoteExist().Result(), res)

	// Check vote is correct
	vote, _ := vm.storage.GetVote(ctx, types.ProposalKey(strconv.FormatInt(proposalID, 10)), "user2")
	assert.Equal(t, true, vote.Result)
	assert.Equal(t, user2, vote.Voter)

	voteList, _ := vm.storage.GetAllVotes(ctx, types.ProposalKey(strconv.FormatInt(proposalID, 10)))
	assert.Equal(t, user3, voteList[1].Voter)

	// test delete vote
	vm.storage.DeleteVote(ctx, types.ProposalKey(strconv.FormatInt(proposalID, 10)), "user2")
	vote, err := vm.storage.GetVote(ctx, types.ProposalKey(strconv.FormatInt(proposalID, 10)), "user2")
	assert.Equal(t, ErrGetVote(), err)

}

func TestDelegatorWithdraw(t *testing.T) {
	ctx, am, vm, gm := setupTest(t, 0)
	user1 := createTestAccount(ctx, am, "user1")
	user2 := createTestAccount(ctx, am, "user2")
	handler := NewHandler(vm, am, gm)

	param, _ := vm.paramHolder.GetVoteParam(ctx)
	vm.AddVoter(ctx, user1, param.VoterMinDeposit)

	cases := []struct {
		addDelegation bool
		delegatedCoin types.Coin
		delegator     types.AccountKey
		voter         types.AccountKey
		withdraw      types.LNO
		expectResult  sdk.Result
	}{
		{false, types.NewCoin(0), user2, user1, "1", ErrIllegalWithdraw().Result()},
		{true, types.NewCoin(100 * types.Decimals), user2, user1, "0.1", ErrIllegalWithdraw().Result()},
		{false, types.NewCoin(0), user2, user1, "101", ErrIllegalWithdraw().Result()},
		{false, types.NewCoin(0), user2, user1, "10", sdk.Result{}},
	}

	for _, cs := range cases {
		if cs.addDelegation {
			vm.AddDelegation(ctx, cs.voter, cs.delegator, cs.delegatedCoin)
		}
		msg := NewDelegatorWithdrawMsg(string(cs.delegator), string(cs.voter), cs.withdraw)
		res := handler(ctx, msg)
		assert.Equal(t, cs.expectResult, res)
	}
}
