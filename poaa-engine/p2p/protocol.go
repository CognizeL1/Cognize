package p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Protocol struct {
	host   host.Host
	topics map[string]*Topic
	mu     sync.RWMutex
	handle map[protocol.ID]MessageHandler
}

type MessageHandler func(ctx context.Context, peerID peer.ID, msg *Message) error

type Message struct {
	Type    MessageType
	Payload []byte
	From    peer.ID
}

type MessageType uint8

const (
	MsgVertex    MessageType = 0x01
	MsgConfirm   MessageType = 0x02
	MsgFinalize  MessageType = 0x03
	MsgSync      MessageType = 0x04
	MsgVote      MessageType = 0x05
	MsgHeartbeat MessageType = 0x06
)

type Topic struct {
	Name     string
	handlers []MessageHandler
	mu       sync.RWMutex
}

func NewProtocol(h host.Host) *Protocol {
	p := &Protocol{
		host:   h,
		topics: make(map[string]*Topic),
		handle: make(map[protocol.ID]MessageHandler),
	}

	h.SetStreamHandler(ProtocolID, p.handleStream)

	return p
}

func (p *Protocol) handleStream(s network.Stream) {
	defer s.Close()

	msg, err := p.readMessage(s)
	if err != nil {
		return
	}

	msg.From = s.Conn().RemotePeer()

	if handler, ok := p.handle[protocol.ID(s.Protocol())]; ok {
		ctx := context.Background()
		handler(ctx, msg.From, msg)
	}
}

func (p *Protocol) readMessage(s network.Stream) (*Message, error) {
	buf := make([]byte, 1024*1024)
	n, err := s.Read(buf)
	if err != nil {
		return nil, err
	}

	if n < 1 {
		return nil, fmt.Errorf("empty message")
	}

	msgType := MessageType(buf[0])
	payload := buf[1:n]

	return &Message{
		Type:    msgType,
		Payload: payload,
	}, nil
}

func (p *Protocol) writeMessage(s network.Stream, msg *Message) error {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(msg.Type))
	buf.Write(msg.Payload)

	_, err := s.Write(buf.Bytes())
	return err
}

func (p *Protocol) SendMessage(ctx context.Context, peerID peer.ID, msgType MessageType, payload []byte) error {
	s, err := p.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()

	msg := &Message{
		Type:    msgType,
		Payload: payload,
	}

	return p.writeMessage(s, msg)
}

func (p *Protocol) Broadcast(ctx context.Context, msgType MessageType, payload []byte) error {
	peers := p.host.Network().Peers()

	for _, pid := range peers {
		if pid == p.host.ID() {
			continue
		}

		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_ = p.SendMessage(ctx, pid, msgType, payload)
		}(pid)
	}

	return nil
}

func (p *Protocol) RegisterHandler(msgType MessageType, handler MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	protoID := protocol.ID(fmt.Sprintf("%s/%d", ProtocolID, msgType))
	p.handle[protoID] = handler
}

func (p *Protocol) Subscribe(topic string, handler MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if t, ok := p.topics[topic]; ok {
		t.mu.Lock()
		t.handlers = append(t.handlers, handler)
		t.mu.Unlock()
	} else {
		p.topics[topic] = &Topic{
			Name:     topic,
			handlers: []MessageHandler{handler},
		}
	}
}

func (p *Protocol) Publish(topic string, msg *Message) error {
	p.mu.RLock()
	t, ok := p.topics[topic]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("topic not found: %s", topic)
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, handler := range t.handlers {
		go handler(context.Background(), p.host.ID(), msg)
	}

	return nil
}

func EncodeVertexPayload(hash [32]byte, parents [2][32]byte, timestamp int64, sender []byte, txBytes []byte) []byte {
	buf := new(bytes.Buffer)
	buf.Write(hash[:])
	for _, p := range parents {
		buf.Write(p[:])
	}
	binary.Write(buf, binary.BigEndian, timestamp)
	binary.Write(buf, binary.BigEndian, uint64(len(sender)))
	buf.Write(sender)
	binary.Write(buf, binary.BigEndian, uint64(len(txBytes)))
	buf.Write(txBytes)
	return buf.Bytes()
}

func DecodeVertexPayload(payload []byte) (hash [32]byte, parents [2][32]byte, timestamp int64, sender []byte, txBytes []byte, err error) {
	if len(payload) < 32+64+8 {
		err = fmt.Errorf("payload too short")
		return
	}

	offset := 0
	copy(hash[:], payload[offset:offset+32])
	offset += 32

	for i := 0; i < 2; i++ {
		copy(parents[i][:], payload[offset:offset+32])
		offset += 32
	}

	timestamp = int64(binary.BigEndian.Uint64(payload[offset : offset+8]))
	offset += 8

	senderLen := binary.BigEndian.Uint64(payload[offset : offset+8])
	offset += 8

	if len(payload) < offset+int(senderLen)+8 {
		err = fmt.Errorf("payload too short for sender")
		return
	}

	sender = make([]byte, senderLen)
	copy(sender, payload[offset:offset+int(senderLen)])
	offset += int(senderLen)

	txLen := binary.BigEndian.Uint64(payload[offset : offset+8])
	offset += 8

	if len(payload) < offset+int(txLen) {
		err = fmt.Errorf("payload too short for tx")
		return
	}

	txBytes = make([]byte, txLen)
	copy(txBytes, payload[offset:offset+int(txLen)])

	return
}

func EncodeConfirmPayload(hash [32]byte, agent []byte, reputation int64, weight []byte) []byte {
	buf := new(bytes.Buffer)
	buf.Write(hash[:])
	binary.Write(buf, binary.BigEndian, uint64(len(agent)))
	buf.Write(agent)
	binary.Write(buf, binary.BigEndian, reputation)
	binary.Write(buf, binary.BigEndian, uint64(len(weight)))
	buf.Write(weight)
	return buf.Bytes()
}

func DecodeConfirmPayload(payload []byte) (hash [32]byte, agent []byte, reputation int64, weight []byte, err error) {
	if len(payload) < 32+8 {
		err = fmt.Errorf("payload too short")
		return
	}

	copy(hash[:], payload[:32])
	offset := 32

	agentLen := binary.BigEndian.Uint64(payload[offset : offset+8])
	offset += 8

	agent = make([]byte, agentLen)
	copy(agent, payload[offset:offset+int(agentLen)])
	offset += int(agentLen)

	reputation = int64(binary.BigEndian.Uint64(payload[offset : offset+8]))
	offset += 8

	weightLen := binary.BigEndian.Uint64(payload[offset : offset+8])
	offset += 8

	weight = make([]byte, weightLen)
	copy(weight, payload[offset:offset+int(weightLen)])

	return
}
