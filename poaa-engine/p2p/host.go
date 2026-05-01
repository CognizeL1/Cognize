package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const (
	ProtocolID   = "/cognize/poaa/1.0.0"
	ServiceTag   = "cognize-poaa"
	DiscoveryTag = "_cognize._udp"
)

type Host struct {
	host   host.Host
	topics map[string]*Topic
	ctx    context.Context
	cancel context.CancelFunc
	config *HostConfig
}

type HostConfig struct {
	Port            int
	BootstrapPeers []string
	EnableMDNS      bool
}

func DefaultHostConfig() *HostConfig {
	return &HostConfig{
		Port:       0,
		EnableMDNS: true,
	}
}

func NewHost(ctx context.Context, config *HostConfig) (*Host, error) {
	if config == nil {
		config = DefaultHostConfig()
	}

	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.Port)),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	hostInstance := &Host{
		host:   h,
		topics: make(map[string]*Topic),
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}

	if config.EnableMDNS {
		hostInstance.startMDNS()
	}

	return hostInstance, nil
}

func (h *Host) startMDNS() {
	mdnsService := mdns.NewMdnsService(h.host, ServiceTag, h)
	if err := mdnsService.Start(); err != nil {
		return
	}
}

func (h *Host) HandlePeerFound(p peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	defer cancel()

	if err := h.host.Connect(ctx, p); err != nil {
		return
	}
}

func (h *Host) ID() peer.ID {
	return h.host.ID()
}

func (h *Host) Addresses() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

func (h *Host) Connect(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	ctx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	defer cancel()

	return h.host.Connect(ctx, *info)
}

func (h *Host) AddBootstrapPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	h.host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return nil
}

func (h *Host) GetPeerInfo() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.host.ID(),
		Addrs: h.host.Addrs(),
	}
}

func (h *Host) Bootstrap(ctx context.Context) error {
	for _, addr := range h.config.BootstrapPeers {
		if err := h.Connect(addr); err != nil {
			continue
		}
	}
	return nil
}

func (h *Host) Close() error {
	h.cancel()
	return h.host.Close()
}

func (h *Host) Host() host.Host {
	return h.host
}
