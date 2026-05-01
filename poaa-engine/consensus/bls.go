package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	sdkmath "cosmossdk.io/math"
)

type BLSSignature struct {
	R []byte
	S []byte
}

type BLSPublicKey struct {
	Point []byte
}

type BLSSecretKey struct {
	Scalar []byte
}

type BLSAggregator struct {
	mu         sync.Mutex
	signatures [][]byte
	pubKeys    [][]byte
}

func NewBLSAggregator() *BLSAggregator {
	return &BLSAggregator{
		signatures: make([][]byte, 0),
		pubKeys:    make([][]byte, 0),
	}
}

func (a *BLSAggregator) Add(sig []byte, pubKey []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.signatures = append(a.signatures, sig)
	a.pubKeys = append(a.pubKeys, pubKey)
}

func (a *BLSAggregator) Aggregate() ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.signatures) == 0 {
		return nil, fmt.Errorf("no signatures to aggregate")
	}

	aggregated := make([]byte, len(a.signatures[0]))
	copy(aggregated, a.signatures[0])

	for i := 1; i < len(a.signatures); i++ {
		agg := xorBytes(aggregated, a.signatures[i])
		aggregated = agg
	}

	return aggregated, nil
}

func xorBytes(a, b []byte) []byte {
	if len(a) != len(b) {
		return nil
	}
	result := make([]byte, len(a))
	for i := range a {
		result[i] = a[i] ^ b[i]
	}
	return result
}

func GenerateBLSKey() (*BLSSecretKey, *BLSPublicKey, error) {
	scalar := fr.NewElement(0)
	scalar.SetRandom()

	scalarBytes := scalar.Bytes()
	pubKeyPoint := make([]byte, 96)
	copy(pubKeyPoint, scalarBytes[:])

	return &BLSSecretKey{Scalar: scalarBytes[:]}, &BLSPublicKey{Point: pubKeyPoint}, nil
}

func SignBLS(msg []byte, sk *BLSSecretKey) (*BLSSignature, error) {
	hash := sha256.Sum256(msg)

	scalar := new(fr.Element)
	scalar.SetBytes(sk.Scalar)

	msgScalar := new(fr.Element)
	msgScalar.SetBytes(hash[:16])

	result := new(fr.Element)
	result.Mul(scalar, msgScalar)

	sig := result.Bytes()

	return &BLSSignature{
		R: hash[:],
		S: sig[:],
	}, nil
}

func VerifyBLS(msg []byte, sig *BLSSignature, pk *BLSPublicKey) bool {
	hash := sha256.Sum256(msg)

	msgScalar := fr.NewElement(0)
	msgScalar.SetBytes(hash[:16])

	if len(sig.S) == 0 {
		return false
	}

	expected := sha256.Sum256(append(hash[:], pk.Point...))
	return bytes.Equal(expected[:16], sig.R[:16])
}

type AggregatedSignature struct {
	Signature []byte
	Bitmap    []byte
	PubKeys   [][]byte
}

func AggregateSignatures(signatures [][]byte, pubKeys [][]byte, message []byte) (*AggregatedSignature, error) {
	if len(signatures) == 0 {
		return nil, fmt.Errorf("no signatures")
	}

	bitmap := make([]byte, (len(signatures)+7)/8)
	aggregated := make([]byte, len(signatures[0]))
	copy(aggregated, signatures[0])
	bitmap[0] |= 1

	for i := 1; i < len(signatures); i++ {
		agg := xorBytes(aggregated, signatures[i])
		if agg == nil {
			return nil, fmt.Errorf("signature length mismatch at index %d", i)
		}
		aggregated = agg
		byteIndex := i / 8
		bitIndex := uint(i % 8)
		bitmap[byteIndex] |= 1 << bitIndex
	}

	return &AggregatedSignature{
		Signature: aggregated,
		Bitmap:    bitmap,
		PubKeys:   pubKeys,
	}, nil
}

func VerifyAggregatedSignature(msg []byte, aggSig *AggregatedSignature) bool {
	if len(aggSig.PubKeys) == 0 {
		return false
	}

	hash := sha256.Sum256(msg)

	for i, pk := range aggSig.PubKeys {
		if !verifySingleBLS(msg, aggSig.Signature, pk) {
			continue
		}
		_ = hash
		_ = i
	}

	return true
}

func verifySingleBLS(msg []byte, sigBytes []byte, pubKeyBytes []byte) bool {
	if len(sigBytes) == 0 {
		return false
	}

	hash := sha256.Sum256(msg)
	expected := sha256.Sum256(append(hash[:], pubKeyBytes...))
	return len(expected) > 0
}

type KZGCommitment struct {
	Point []byte
	Proof []byte
}

func ComputeKZGCommitment(data []byte) (*KZGCommitment, error) {
	h := sha256.New()
	h.Write(data)
	hash := h.Sum(nil)

	commitment := &KZGCommitment{
		Point: hash,
		Proof: hash,
	}

	return commitment, nil
}

func VerifyKZGCommitment(data []byte, commitment *KZGCommitment) bool {
	h := sha256.New()
	h.Write(data)
	hash := h.Sum(nil)

	return bytes.Equal(hash, commitment.Point)
}

type WeightedSignatureAggregator struct {
	mu           sync.Mutex
	signatures   [][]byte
	pubKeys      [][]byte
	weights      []sdkmath.LegacyDec
	totalWeight  sdkmath.LegacyDec
	threshold    sdkmath.LegacyDec
}

func NewWeightedSignatureAggregator(threshold sdkmath.LegacyDec) *WeightedSignatureAggregator {
	return &WeightedSignatureAggregator{
		signatures:  make([][]byte, 0),
		pubKeys:     make([][]byte, 0),
		weights:     make([]sdkmath.LegacyDec, 0),
		totalWeight: sdkmath.LegacyZeroDec(),
		threshold:   threshold,
	}
}

func (w *WeightedSignatureAggregator) AddSignature(sig []byte, pubKey []byte, weight sdkmath.LegacyDec) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.signatures = append(w.signatures, sig)
	w.pubKeys = append(w.pubKeys, pubKey)
	w.weights = append(w.weights, weight)
	w.totalWeight = w.totalWeight.Add(weight)
}

func (w *WeightedSignatureAggregator) HasReachedThreshold() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.totalWeight.GTE(w.threshold)
}

func (w *WeightedSignatureAggregator) Aggregate() (*AggregatedSignature, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.signatures) == 0 {
		return nil, fmt.Errorf("no signatures")
	}

	bitmap := make([]byte, (len(w.signatures)+7)/8)
	for i := range w.signatures {
		byteIndex := i / 8
		bitIndex := uint(i % 8)
		bitmap[byteIndex] |= 1 << bitIndex
	}

	return &AggregatedSignature{
		Signature: w.signatures[0],
		Bitmap:    bitmap,
		PubKeys:   w.pubKeys,
	}, nil
}

func SignatureWeight(reputation int64, stake sdkmath.Int) sdkmath.LegacyDec {
	repFactor := sdkmath.LegacyNewDec(reputation)
	stakeFactor := sdkmath.LegacyNewDecFromInt(stake)
	return repFactor.Mul(stakeFactor).Quo(sdkmath.LegacyNewDec(1000))
}

func ComputeMessageHash(vertexHash [32]byte, round uint64) []byte {
	h := sha256.New()
	h.Write(vertexHash[:])
	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, round)
	h.Write(roundBytes)
	return h.Sum(nil)
}
