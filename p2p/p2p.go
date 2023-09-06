package p2p

import (
	"context"
	"fmt"
	"os"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	remotesapi "github.com/dolthub/dolt/go/gen/proto/dolt/services/remotesapi/v1alpha1"
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
	"github.com/protosio/distributeddolt/db"
	dbclient "github.com/protosio/distributeddolt/db/client"
	pinger "github.com/protosio/distributeddolt/p2p/server"
	"github.com/protosio/distributeddolt/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	protosRPCProtocol = protocol.ID("/protos/rpc/0.0.1")
)

type P2PClient struct {
	proto.PingerClient
	proto.DBSyncerClient
	remotesapi.ChunkStoreServiceClient
	proto.DownloaderClient

	id string
}

func (c *P2PClient) GetID() string {
	return c.id
}

type P2P struct {
	peerHandler  db.PeerHandler
	log          *logrus.Logger
	host         host.Host
	grpcServer   *grpc.Server
	PeerChan     chan peer.AddrInfo
	peerListChan chan peer.IDSlice
	clients      cmap.ConcurrentMap
}

func (p2p *P2P) HandlePeerFound(pi peer.AddrInfo) {
	p2p.PeerChan <- pi
}

func (p2p *P2P) GetClient(id string) (dbclient.Client, error) {
	clientIface, found := p2p.clients.Get(id)
	if !found {
		return nil, fmt.Errorf("client %s not found", id)
	}
	client, ok := clientIface.(*P2PClient)
	if !ok {
		return nil, fmt.Errorf("client %s not found", id)
	}
	return client, nil
}

func (p2p *P2P) GetClients() []dbclient.Client {
	clients := make([]dbclient.Client, 0)
	for clientIface := range p2p.clients.IterBuffered() {
		client, ok := clientIface.Val.(*P2PClient)
		if !ok {
			panic(fmt.Errorf("client %s has incorrect type", clientIface.Key))
		}

		clients = append(clients, client)
	}
	return clients
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
				dbSyncerClient := proto.NewDBSyncerClient(conn)
				downloaderClient := proto.NewDownloaderClient(conn)
				csClient := remotesapi.NewChunkStoreServiceClient(conn)
				client := &P2PClient{
					PingerClient:            pingerClient,
					DBSyncerClient:          dbSyncerClient,
					ChunkStoreServiceClient: csClient,
					DownloaderClient:        downloaderClient,
					id:                      peer.ID.String(),
				}

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
				if p2p.peerHandler != nil {
					err = p2p.peerHandler.AddPeer(peer.ID.String())
					if err != nil {
						p2p.log.Errorf("Failed to add DB remote for '%s': %v", peer.ID.String(), err)
					}
				}
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
		p2p.log.Errorf("Error while disconnecting from peer '%s': %v", conn.RemotePeer().String(), err)
	}
	p2p.clients.Remove(conn.RemotePeer().String())
	if p2p.peerHandler != nil {
		if err := p2p.peerHandler.RemovePeer(conn.RemotePeer().String()); err != nil {
			p2p.log.Errorf("Failed to remove DB peer for '%s': %v", conn.RemotePeer().String(), err)
		}
	}
}

func (p2p *P2P) GetGRPCServer() *grpc.Server {
	return p2p.grpcServer
}

func (p2p *P2P) RegisterPeerHandler(handler db.PeerHandler) {
	p2p.peerHandler = handler
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer() (func() error, error) {

	p2p.log.Infof("Starting p2p server using id %s", p2p.host.ID())
	ctx := context.TODO()

	// register internal grpc servers
	proto.RegisterPingerServer(p2p.grpcServer, &pinger.Server{})

	// serve grpc server over libp2p host
	grpcListener := p2pgrpc.NewListener(ctx, p2p.host, protosRPCProtocol)
	go func() {
		err := p2p.grpcServer.Serve(grpcListener)
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
		p2p.grpcServer.GracefulStop()
		return p2p.host.Close()
	}

	return stopper, nil

}

// NewManager creates and returns a new p2p manager
func NewManager(workdir string, port int, peerListChan chan peer.IDSlice, logger *logrus.Logger) (*P2P, error) {
	p2p := &P2P{
		PeerChan:     make(chan peer.AddrInfo),
		peerListChan: peerListChan,
		clients:      cmap.New(),
		log:          logger,
		grpcServer:   grpc.NewServer(p2pgrpc.WithP2PCredentials()),
	}

	var prvKey crypto.PrivKey
	keyFile := workdir + "/key"
	fileInfo, err := os.Stat(keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			prvKey, _, err = crypto.GenerateKeyPair(crypto.Ed25519, 0)
			if err != nil {
				return nil, err
			}
			prvKeyBytes, err := crypto.MarshalPrivateKey(prvKey)
			if err != nil {
				return nil, err
			}
			err = os.WriteFile(keyFile, prvKeyBytes, 0600)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		if fileInfo.IsDir() {
			return nil, fmt.Errorf("key file %s is a directory", keyFile)
		}
		prvKeyBytes, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		prvKey, err = crypto.UnmarshalPrivateKey(prvKeyBytes)
		if err != nil {
			return nil, err
		}
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
