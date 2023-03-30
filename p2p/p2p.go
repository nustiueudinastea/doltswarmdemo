package p2p

import (
	"context"
	"fmt"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/protosio/testdolt/pinger"
	"github.com/protosio/testdolt/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	protosRPCProtocol = protocol.ID("/protos/rpc/0.0.1")
	// protosUpdatesTopic            = protocol.ID("/protos/updates/0.0.1")
)

type Client struct {
	proto.PingerClient
}

type P2P struct {
	log             *logrus.Logger
	host            host.Host
	PeerChan        chan peer.AddrInfo
	peerListChan    chan peer.IDSlice
	clients         cmap.ConcurrentMap
	broadcastClient *BroadcastClient
}

func (p2p *P2P) HandlePeerFound(pi peer.AddrInfo) {
	p2p.PeerChan <- pi
}

func (p2p *P2P) GetClient(id peer.ID) (*Client, error) {
	clientIface, found := p2p.clients.Get(id.String())
	if !found {
		return nil, fmt.Errorf("Client %s not found", id.String())
	}
	client, ok := clientIface.(*Client)
	if !ok {
		return nil, fmt.Errorf("Client %s not found", id.String())
	}
	return client, nil
}

func (p2p *P2P) GetBroadcastClient() *BroadcastClient {
	return p2p.broadcastClient
}

func (p2p *P2P) peerDiscoveryProcessor() func() error {
	stopSignal := make(chan struct{})
	go func() {
		p2p.log.Info("Starting peer discovery processor")
		for {
			select {
			case peer := <-p2p.PeerChan:
				p2p.log.Infof("New peer. Connecting: %s", peer)
				ctx := context.Background()
				if err := p2p.host.Connect(ctx, peer); err != nil {
					p2p.log.Error("Connection failed: ", err)
					continue
				}

				tries := 0
				for {
					if tries == 5 {
						break
					}
					tries += 1

					if p2p.host.Network().Connectedness(peer.ID) != network.Connected {
						p2p.log.Infof("Waiting for peer connection with %s(%s)", peer.ID.String(), p2p.host.Network().Connectedness(peer.ID))
						time.Sleep(1 * time.Second)
						continue
					} else {
						break
					}

				}

				if p2p.host.Network().Connectedness(peer.ID) != network.Connected {
					p2p.log.Errorf("Connection to %s failed", peer.ID.String())
					continue
				}

				// grpc conn
				conn, err := grpc.Dial(
					peer.ID.String(),
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					p2pgrpc.WithP2PDialer(p2p.host, protosRPCProtocol),
				)
				if err != nil {
					p2p.log.Error("Grpc conn failed: ", err)
					continue
				}

				// client
				pingerClient := proto.NewPingerClient(conn)
				client := &Client{pingerClient}

				// test connectivity with a ping
				_, err = client.Ping(ctx, &proto.PingRequest{
					Ping: "pong",
				})
				if err != nil {
					p2p.log.Error("Ping failed: ", err)
					continue
				}

				p2p.log.Infof("Connected to %s", peer.ID.String())
				p2p.clients.Set(peer.ID.String(), client)
				p2p.peerListChan <- p2p.host.Network().Peers()

			case <-stopSignal:
				p2p.log.Info("Stopping peer discovery processor")
				return
			}
		}
	}()
	stopper := func() error {
		stopSignal <- struct{}{}
		return nil
	}
	return stopper
}

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {
	p2p.log.Infof("Disconnected from %s", conn.RemotePeer().String())
	p2p.peerListChan <- p2p.host.Network().Peers()
	if err := conn.Close(); err != nil {
		p2p.log.Error("Error while disconnecting from peer '%s': %v", conn.RemotePeer().String(), err)
	}
	p2p.clients.Remove(conn.RemotePeer().String())
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer() (func() error, error) {

	p2p.log.Infof("Starting p2p server using id %s", p2p.host.ID())

	ctx := context.TODO()

	grpcServer := grpc.NewServer(p2pgrpc.WithP2PCredentials())
	proto.RegisterPingerServer(grpcServer, &pinger.Server{})

	// serve grpc server over libp2p host
	grpcListener := p2pgrpc.NewListener(ctx, p2p.host, protosRPCProtocol)
	go func() {
		err := grpcServer.Serve(grpcListener)
		if err != nil {
			p2p.log.Error("grpc serve error: ", err)
			panic(err)
		}
	}()

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %w", err)
	}

	peerDiscoveryStopper := p2p.peerDiscoveryProcessor()

	mdnsService := mdns.NewMdnsService(p2p.host, "protos", p2p)
	if err := mdnsService.Start(); err != nil {
		panic(err)
	}

	stopper := func() error {
		p2p.log.Debug("Stopping p2p server")
		peerDiscoveryStopper()
		mdnsService.Close()
		grpcServer.GracefulStop()
		return p2p.host.Close()
	}

	return stopper, nil

}

// NewManager creates and returns a new p2p manager
func NewManager(initMode bool, port int, peerListChan chan peer.IDSlice, logger *logrus.Logger) (*P2P, error) {
	p2p := &P2P{
		PeerChan:     make(chan peer.AddrInfo),
		peerListChan: peerListChan,
		clients:      cmap.New(),
		log:          logger,
	}

	p2p.broadcastClient = &BroadcastClient{p2p: p2p}

	prvKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, err
	}

	con, err := connmgr.NewConnManager(100, 400)
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(con),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %w", err)
	}

	p2p.host = host
	nb := network.NotifyBundle{
		DisconnectedF: p2p.closeConnectionHandler,
	}
	p2p.host.Network().Notify(&nb)

	p2p.log.Debugf("Using host with ID '%s'", host.ID().String())
	return p2p, nil
}
