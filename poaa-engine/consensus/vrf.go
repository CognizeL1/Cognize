package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"
)

type VRFProof struct {
	Proof []byte
	Hash  []byte
}

type VRFSecretKey struct {
	key *btcec.PrivateKey
}

type VRFPublicKey struct {
	key *btcec.PublicKey
}

func GenerateVRFKey() (*VRFSecretKey, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate VRF key: %w", err)
	}
	return &VRFSecretKey{key: privKey}, nil
}

func VRFKeyFromBytes(privKeyBytes []byte) (*VRFSecretKey, error) {
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	if privKey == nil {
		return nil, fmt.Errorf("invalid private key bytes")
	}
	return &VRFSecretKey{key: privKey}, nil
}

func (sk *VRFSecretKey) Public() *VRFPublicKey {
	return &VRFPublicKey{key: sk.key.PubKey()}
}

func (sk *VRFSecretKey) Bytes() []byte {
	return sk.key.Serialize()
}

func (sk *VRFSecretKey) SignVRF(message []byte) (*VRFProof, error) {
	hash := sha256.Sum256(message)
	sig := ecdsa.Sign(sk.key, hash[:])

	proof := &VRFProof{
		Proof: sig.Serialize(),
		Hash:  computeVRFHash(sig.Serialize(), sk.Public().key),
	}
	return proof, nil
}

func (pk *VRFPublicKey) VerifyVRF(message []byte, proof *VRFProof) bool {
	hash := sha256.Sum256(message)
	sig, err := ecdsa.ParseSignature(proof.Proof)
	if err != nil {
		return false
	}

	valid := sig.Verify(hash[:], pk.key)
	if !valid {
		return false
	}

	expectedHash := computeVRFHash(proof.Proof, pk.key)
	return bytes.Equal(expectedHash, proof.Hash)
}

func (pk *VRFPublicKey) Bytes() []byte {
	return pk.key.SerializeCompressed()
}

func computeVRFHash(proof []byte, pubKey *btcec.PublicKey) []byte {
	h := sha256.New()
	h.Write(proof)
	h.Write(pubKey.SerializeCompressed())
	return h.Sum(nil)
}

type ValidatorSet struct {
	Validators []Validator
	TotalPower sdkmath.LegacyDec
}

type Validator struct {
	Address    sdk.AccAddress
	Power      sdkmath.LegacyDec
	VRFPubKey  *VRFPublicKey
	Reputation int64
}

func NewValidatorSet() *ValidatorSet {
	return &ValidatorSet{
		Validators: make([]Validator, 0),
		TotalPower: sdkmath.LegacyZeroDec(),
	}
}

func (vs *ValidatorSet) AddValidator(v Validator) {
	vs.Validators = append(vs.Validators, v)
	vs.TotalPower = vs.TotalPower.Add(v.Power)
}

func (vs *ValidatorSet) RemoveValidator(address sdk.AccAddress) {
	for i, v := range vs.Validators {
		if v.Address.Equals(address) {
			vs.TotalPower = vs.TotalPower.Sub(v.Power)
			vs.Validators = append(vs.Validators[:i], vs.Validators[i+1:]...)
			return
		}
	}
}

func (vs *ValidatorSet) SelectProposer(seed []byte, round uint64) (*Validator, error) {
	if len(vs.Validators) == 0 {
		return nil, fmt.Errorf("no validators in set")
	}

	bestHash := make([]byte, 32)
	for i := range bestHash {
		bestHash[i] = 0xFF
	}
	var bestValidator *Validator

	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, round)

	for i, v := range vs.Validators {
		msg := make([]byte, 0)
		msg = append(msg, seed...)
		msg = append(msg, roundBytes...)
		msg = append(msg, v.Address...)
		msg = append(msg, v.Power.BigInt().Bytes()...)

		hash := sha256.Sum256(msg)
		if bytes.Compare(hash[:], bestHash) < 0 {
			copy(bestHash, hash[:])
			bestValidator = &vs.Validators[i]
		}
	}

	if bestValidator == nil {
		return &vs.Validators[0], nil
	}
	return bestValidator, nil
}

func (vs *ValidatorSet) GetValidator(address sdk.AccAddress) (*Validator, bool) {
	for _, v := range vs.Validators {
		if v.Address.Equals(address) {
			return &v, true
		}
	}
	return nil, false
}

func (vs *ValidatorSet) Size() int {
	return len(vs.Validators)
}

func ComputeVRFSeed(prevHash [32]byte, round uint64, timestamp int64) [32]byte {
	h := sha256.New()
	h.Write(prevHash[:])
	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, round)
	h.Write(roundBytes)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(timestamp))
	h.Write(tsBytes)
	return [32]byte(h.Sum(nil))
}
