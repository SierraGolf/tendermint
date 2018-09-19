package types

import (
	"bytes"
	"fmt"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

// PrivValidator defines the functionality of a local Tendermint validator
// that signs votes, proposals, and heartbeats, and never double signs.
type PrivValidator interface {
	GetAddress() Address // redundant since .PubKey().Address()
	GetPubKey() crypto.PubKey

	SignVote(vote *Vote) (SignVoteReply, error)
	SignProposal(chainID string, proposal *Proposal) error
	SignHeartbeat(chainID string, heartbeat *Heartbeat) error
}

//----------------------------------------
// Misc.

type PrivValidatorsByAddress []PrivValidator

func (pvs PrivValidatorsByAddress) Len() int {
	return len(pvs)
}

func (pvs PrivValidatorsByAddress) Less(i, j int) bool {
	return bytes.Compare(pvs[i].GetAddress(), pvs[j].GetAddress()) == -1
}

func (pvs PrivValidatorsByAddress) Swap(i, j int) {
	it := pvs[i]
	pvs[i] = pvs[j]
	pvs[j] = it
}

//----------------------------------------
// MockPV

// MockPV implements PrivValidator without any safety or persistence.
// Only use it for testing.
type MockPV struct {
	privKey crypto.PrivKey
}

func NewMockPV() *MockPV {
	return &MockPV{ed25519.GenPrivKey()}
}

// Implements PrivValidator.
func (pv *MockPV) GetAddress() Address {
	return pv.privKey.PubKey().Address()
}

// Implements PrivValidator.
func (pv *MockPV) GetPubKey() crypto.PubKey {
	return pv.privKey.PubKey()
}

// Implements PrivValidator.
func (pv *MockPV) SignVote(vote *Vote) (SignVoteReply, error) {
	signBytes := vote.SignBytes()
	sig, err := pv.privKey.Sign(signBytes)
	if err != nil {
		// TODO(ismail): encapsulate error into reply!
		return SignVoteReply{}, err
	}

	return SignVoteReply{
		Vote:      *vote,
		Signature: sig,
	}, nil
}

// Implements PrivValidator.
func (pv *MockPV) SignProposal(chainID string, proposal *Proposal) error {
	signBytes := proposal.SignBytes(chainID)
	sig, err := pv.privKey.Sign(signBytes)
	if err != nil {
		return err
	}
	proposal.Signature = sig
	return nil
}

// signHeartbeat signs the heartbeat without any checking.
func (pv *MockPV) SignHeartbeat(chainID string, heartbeat *Heartbeat) error {
	sig, err := pv.privKey.Sign(heartbeat.SignBytes(chainID))
	if err != nil {
		return err
	}
	heartbeat.Signature = sig
	return nil
}

// String returns a string representation of the MockPV.
func (pv *MockPV) String() string {
	return fmt.Sprintf("MockPV{%v}", pv.GetAddress())
}

// XXX: Implement.
func (pv *MockPV) DisableChecks() {
	// Currently this does nothing,
	// as MockPV has no safety checks at all.
}
