package dag

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
)

var (
	ErrVertexNotFound = fmt.Errorf("vertex not found")
	ErrInvalidHash    = fmt.Errorf("invalid hash")
)

type Context struct {
	Height    uint64
	Timestamp int64
	Proposer  []byte
}

type ConfirmationLayer uint8

const (
	LayerSoft    ConfirmationLayer = 0
	LayerFast    ConfirmationLayer = 1
	LayerHard    ConfirmationLayer = 2
	LayerArchive ConfirmationLayer = 3
)

type Vertex struct {
	Hash        [32]byte
	Parents     [2][32]byte
	Timestamp   int64
	Index       uint64

	TxBytes     []byte
	Sender      string
	TxValue     sdkmath.Int

	Layer       ConfirmationLayer
	Confirmed   bool
	AggSig      []byte
	Confirmers  []ConfirmRecord
	TotalWeight sdkmath.LegacyDec
	FinalityAt  int64

	Children    [][32]byte
	Depth       uint64
}

type ConfirmRecord struct {
	Agent      string
	Reputation int64
	Weight      sdkmath.LegacyDec
	Timestamp  int64
}

type TipsPool struct {
	Tips    map[[32]byte]*Vertex
	MaxSize int
}

func NewTipsPool() *TipsPool {
	return &TipsPool{
		Tips:    make(map[[32]byte]*Vertex),
		MaxSize: 1000,
	}
}

func (t *TipsPool) Add(v *Vertex) {
	if len(t.Tips) >= t.MaxSize {
		for k := range t.Tips {
			delete(t.Tips, k)
			break
		}
	}
	t.Tips[v.Hash] = v
}

func (t *TipsPool) Remove(hash [32]byte) {
	delete(t.Tips, hash)
}

func (t *TipsPool) GetAll() []*Vertex {
	result := make([]*Vertex, 0, len(t.Tips))
	for _, v := range t.Tips {
		result = append(result, v)
	}
	return result
}

func (v *Vertex) ComputeHash() [32]byte {
	h := sha256.New()
	h.Write(v.TxBytes)
	for _, p := range v.Parents {
		h.Write(p[:])
	}
	h.Write([]byte(v.Sender))
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v.TxValue.Uint64())
	h.Write(b)
	binary.BigEndian.PutUint64(b, uint64(v.Timestamp))
	h.Write(b)
	return [32]byte(h.Sum(nil))
}

func (v *Vertex) Validate() error {
	if v.Hash != v.ComputeHash() {
		return fmt.Errorf("invalid hash")
	}
	if v.TxBytes == nil {
		return fmt.Errorf("empty tx bytes")
	}
	if v.Layer > LayerArchive {
		return fmt.Errorf("invalid layer")
	}
	return nil
}

func (v *Vertex) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(v.Hash[:])
	for _, p := range v.Parents {
		buf.Write(p[:])
	}
	binary.Write(buf, binary.BigEndian, v.Timestamp)
	binary.Write(buf, binary.BigEndian, v.Index)
	txLen := uint64(len(v.TxBytes))
	binary.Write(buf, binary.BigEndian, txLen)
	buf.Write(v.TxBytes)
	senderBytes := []byte(v.Sender)
	senderLen := uint64(len(senderBytes))
	binary.Write(buf, binary.BigEndian, senderLen)
	buf.Write(senderBytes)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v.TxValue.Uint64()))
	buf.Write(b)
	buf.WriteByte(byte(v.Layer))
	buf.WriteByte(boolToByte(v.Confirmed))
	weightBigInt := v.TotalWeight.BigInt()
	weightBytes := weightBigInt.Bytes()
	weightLen := uint64(len(weightBytes))
	binary.Write(buf, binary.BigEndian, weightLen)
	buf.Write(weightBytes)
	binary.Write(buf, binary.BigEndian, v.FinalityAt)
	binary.Write(buf, binary.BigEndian, v.Depth)
	confCount := uint64(len(v.Confirmers))
	binary.Write(buf, binary.BigEndian, confCount)
	for _, c := range v.Confirmers {
		agentBytes := []byte(c.Agent)
		agentLen := uint64(len(agentBytes))
		binary.Write(buf, binary.BigEndian, agentLen)
		buf.Write(agentBytes)
		binary.Write(buf, binary.BigEndian, c.Reputation)
		weightBigInt := c.Weight.BigInt()
		weightBytes := weightBigInt.Bytes()
		weightLen := uint64(len(weightBytes))
		binary.Write(buf, binary.BigEndian, weightLen)
		buf.Write(weightBytes)
		binary.Write(buf, binary.BigEndian, c.Timestamp)
	}
	childCount := uint64(len(v.Children))
	binary.Write(buf, binary.BigEndian, childCount)
	for _, c := range v.Children {
		buf.Write(c[:])
	}
	return buf.Bytes(), nil
}

func (v *Vertex) FromBytes(data []byte) error {
	if len(data) < 32+64+8+8 {
		return fmt.Errorf("insufficient data")
	}
	offset := 0
	copy(v.Hash[:], data[offset:offset+32])
	offset += 32
	for i := 0; i < 2; i++ {
		copy(v.Parents[i][:], data[offset:offset+32])
		offset += 32
	}
	v.Timestamp = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8
	v.Index = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	txLen := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	if len(data) < offset+int(txLen)+8 {
		return fmt.Errorf("insufficient data for tx")
	}
	v.TxBytes = make([]byte, txLen)
	copy(v.TxBytes, data[offset:offset+int(txLen)])
	offset += int(txLen)
	senderLen := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	if len(data) < offset+int(senderLen)+8 {
		return fmt.Errorf("insufficient data for sender")
	}
	v.Sender = string(data[offset : offset+int(senderLen)])
	offset += int(senderLen)
	txVal := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	v.TxValue = sdkmath.NewIntFromUint64(txVal)
	v.Layer = ConfirmationLayer(data[offset])
	offset++
	v.Confirmed = byteToBool(data[offset])
	offset++
	weightLen := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	if len(data) < offset+int(weightLen)+8+8 {
		return fmt.Errorf("insufficient data for weight")
	}
	weightBytes := make([]byte, weightLen)
	copy(weightBytes, data[offset:offset+int(weightLen)])
	weightInt := new(big.Int).SetBytes(weightBytes)
	v.TotalWeight = sdkmath.LegacyNewDecFromBigInt(weightInt)
	offset += int(weightLen)
	v.FinalityAt = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8
	v.Depth = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	confCount := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	v.Confirmers = make([]ConfirmRecord, confCount)
	for i := uint64(0); i < confCount; i++ {
		agentLen := binary.BigEndian.Uint64(data[offset : offset+8])
		offset += 8
		v.Confirmers[i].Agent = string(data[offset : offset+int(agentLen)])
		offset += int(agentLen)
		v.Confirmers[i].Reputation = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
		offset += 8
		weightLen := binary.BigEndian.Uint64(data[offset : offset+8])
		offset += 8
		weightBytes := make([]byte, weightLen)
		copy(weightBytes, data[offset:offset+int(weightLen)])
		weightInt := new(big.Int).SetBytes(weightBytes)
		v.Confirmers[i].Weight = sdkmath.LegacyNewDecFromBigInt(weightInt)
		offset += int(weightLen)
		v.Confirmers[i].Timestamp = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
		offset += 8
	}
	childCount := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8
	v.Children = make([][32]byte, childCount)
	for i := uint64(0); i < childCount; i++ {
		copy(v.Children[i][:], data[offset:offset+32])
		offset += 32
	}
	return nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func byteToBool(b byte) bool {
	return b != 0
}

func GetCurrentTimestamp() int64 {
	return 0
}
