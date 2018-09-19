package types

import (
	"bytes"
	"fmt"

	"github.com/tendermint/go-amino"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/merkle"
)

const (
	// MaxEvidenceBytes is a maximum size of any evidence (including amino overhead).
	MaxEvidenceBytes = 440
)

// ErrEvidenceInvalid wraps a piece of evidence and the error denoting how or why it is invalid.
type ErrEvidenceInvalid struct {
	Evidence   Evidence
	ErrorValue error
}

func NewEvidenceInvalidErr(ev Evidence, err error) *ErrEvidenceInvalid {
	return &ErrEvidenceInvalid{ev, err}
}

// Error returns a string representation of the error.
func (err *ErrEvidenceInvalid) Error() string {
	return fmt.Sprintf("Invalid evidence: %v. Evidence: %v", err.ErrorValue, err.Evidence)
}

//-------------------------------------------

// Evidence represents any provable malicious activity by a validator
type Evidence interface {
	Height() int64                                     // height of the equivocation
	Address() []byte                                   // address of the equivocating validator
	Hash() []byte                                      // hash of the evidence
	Verify(chainID string, pubKey crypto.PubKey) error // verify the evidence
	Equal(Evidence) bool                               // check equality of evidence

	String() string
}

func RegisterEvidences(cdc *amino.Codec) {
	cdc.RegisterInterface((*Evidence)(nil), nil)
	cdc.RegisterConcrete(&DuplicateVoteEvidence{}, "tendermint/DuplicateVoteEvidence", nil)

	// mocks
	cdc.RegisterConcrete(MockGoodEvidence{}, "tendermint/MockGoodEvidence", nil)
	cdc.RegisterConcrete(MockBadEvidence{}, "tendermint/MockBadEvidence", nil)
}

// MaxEvidenceBytesPerBlock returns the maximum evidence size per block.
func MaxEvidenceBytesPerBlock(blockMaxBytes int) int {
	return blockMaxBytes / 10
}

//-------------------------------------------

// DuplicateVoteEvidence contains evidence a validator signed two conflicting votes.
type DuplicateVoteEvidence struct {
	PubKey crypto.PubKey
	// TODO(ismail): this probably need to be `SignedVoteReply`s
	VoteA *SignVoteReply
	VoteB *SignVoteReply
}

// String returns a string representation of the evidence.
func (dve *DuplicateVoteEvidence) String() string {
	return fmt.Sprintf("VoteA: %v; VoteB: %v", dve.VoteA, dve.VoteB)

}

// Height returns the height this evidence refers to.
func (dve *DuplicateVoteEvidence) Height() int64 {
	return dve.VoteA.Vote.Height
}

// Address returns the address of the validator.
func (dve *DuplicateVoteEvidence) Address() []byte {
	return dve.PubKey.Address()
}

// Hash returns the hash of the evidence.
func (dve *DuplicateVoteEvidence) Hash() []byte {
	return aminoHasher(dve).Hash()
}

// Verify returns an error if the two votes aren't conflicting.
// To be conflicting, they must be from the same validator, for the same H/R/S, but for different blocks.
func (dve *DuplicateVoteEvidence) Verify(chainID string, pubKey crypto.PubKey) error {
	// H/R/S must be the same
	if dve.VoteA.Vote.Height != dve.VoteB.Vote.Height ||
		dve.VoteA.Vote.Round != dve.VoteB.Vote.Round ||
		dve.VoteA.Vote.Type != dve.VoteB.Vote.Type {
		return fmt.Errorf("DuplicateVoteEvidence Error: H/R/S does not match. Got %v and %v", dve.VoteA, dve.VoteB)
	}

	// Address must be the same
	if !bytes.Equal(dve.VoteA.Vote.ValidatorAddress, dve.VoteB.Vote.ValidatorAddress) {
		return fmt.Errorf("DuplicateVoteEvidence Error: Validator addresses do not match. Got %X and %X", dve.VoteA.Vote.ValidatorAddress, dve.VoteB.Vote.ValidatorAddress)
	}

	// Index must be the same
	if dve.VoteA.Vote.ValidatorIndex != dve.VoteB.Vote.ValidatorIndex {
		return fmt.Errorf("DuplicateVoteEvidence Error: Validator indices do not match. Got %d and %d", dve.VoteA.Vote.ValidatorIndex, dve.VoteB.Vote.ValidatorIndex)
	}

	// BlockIDs must be different
	if dve.VoteA.Vote.BlockID.Equals(dve.VoteB.Vote.BlockID) {
		return fmt.Errorf("DuplicateVoteEvidence Error: BlockIDs are the same (%v) - not a real duplicate vote", dve.VoteA.Vote.BlockID)
	}

	// pubkey must match address (this should already be true, sanity check)
	addr := dve.VoteA.Vote.ValidatorAddress
	if !bytes.Equal(pubKey.Address(), addr) {
		return fmt.Errorf("DuplicateVoteEvidence FAILED SANITY CHECK - address (%X) doesn't match pubkey (%v - %X)",
			addr, pubKey, pubKey.Address())
	}

	// Signatures must be valid
	if !pubKey.VerifyBytes(dve.VoteA.Vote.SignBytes(), dve.VoteA.Signature) {
		return fmt.Errorf("DuplicateVoteEvidence Error verifying VoteA: %v", ErrVoteInvalidSignature)
	}
	if !pubKey.VerifyBytes(dve.VoteB.Vote.SignBytes(), dve.VoteB.Signature) {
		return fmt.Errorf("DuplicateVoteEvidence Error verifying VoteB: %v", ErrVoteInvalidSignature)
	}

	return nil
}

// Equal checks if two pieces of evidence are equal.
func (dve *DuplicateVoteEvidence) Equal(ev Evidence) bool {
	if _, ok := ev.(*DuplicateVoteEvidence); !ok {
		return false
	}

	// just check their hashes
	dveHash := aminoHasher(dve).Hash()
	evHash := aminoHasher(ev).Hash()
	return bytes.Equal(dveHash, evHash)
}

//-----------------------------------------------------------------

// UNSTABLE
type MockGoodEvidence struct {
	Height_  int64
	Address_ []byte
}

// UNSTABLE
func NewMockGoodEvidence(height int64, idx int, address []byte) MockGoodEvidence {
	return MockGoodEvidence{height, address}
}

func (e MockGoodEvidence) Height() int64   { return e.Height_ }
func (e MockGoodEvidence) Address() []byte { return e.Address_ }
func (e MockGoodEvidence) Hash() []byte {
	return []byte(fmt.Sprintf("%d-%x", e.Height_, e.Address_))
}
func (e MockGoodEvidence) Verify(chainID string, pubKey crypto.PubKey) error { return nil }
func (e MockGoodEvidence) Equal(ev Evidence) bool {
	e2 := ev.(MockGoodEvidence)
	return e.Height_ == e2.Height_ &&
		bytes.Equal(e.Address_, e2.Address_)
}
func (e MockGoodEvidence) String() string {
	return fmt.Sprintf("GoodEvidence: %d/%s", e.Height_, e.Address_)
}

// UNSTABLE
type MockBadEvidence struct {
	MockGoodEvidence
}

func (e MockBadEvidence) Verify(chainID string, pubKey crypto.PubKey) error {
	return fmt.Errorf("MockBadEvidence")
}
func (e MockBadEvidence) Equal(ev Evidence) bool {
	e2 := ev.(MockBadEvidence)
	return e.Height_ == e2.Height_ &&
		bytes.Equal(e.Address_, e2.Address_)
}
func (e MockBadEvidence) String() string {
	return fmt.Sprintf("BadEvidence: %d/%s", e.Height_, e.Address_)
}

//-------------------------------------------

// EvidenceList is a list of Evidence. Evidences is not a word.
type EvidenceList []Evidence

// Hash returns the simple merkle root hash of the EvidenceList.
func (evl EvidenceList) Hash() []byte {
	// Recursive impl.
	// Copied from crypto/merkle to avoid allocations
	switch len(evl) {
	case 0:
		return nil
	case 1:
		return evl[0].Hash()
	default:
		left := EvidenceList(evl[:(len(evl)+1)/2]).Hash()
		right := EvidenceList(evl[(len(evl)+1)/2:]).Hash()
		return merkle.SimpleHashFromTwoHashes(left, right)
	}
}

func (evl EvidenceList) String() string {
	s := ""
	for _, e := range evl {
		s += fmt.Sprintf("%s\t\t", e)
	}
	return s
}

// Has returns true if the evidence is in the EvidenceList.
func (evl EvidenceList) Has(evidence Evidence) bool {
	for _, ev := range evl {
		if ev.Equal(evidence) {
			return true
		}
	}
	return false
}
