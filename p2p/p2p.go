package p2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
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
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/martinlindhe/base36"
	p2pproto "github.com/nustiueudinastea/doltswarmdemo/p2p/proto"
	p2psrv "github.com/nustiueudinastea/doltswarmdemo/p2p/server"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	protosRPCProtocol = protocol.ID("/protos/rpc/0.0.1")
)

type P2PClient struct {
	p2pproto.PingerClient
	p2pproto.TesterClient

	id string
}

func (c *P2PClient) GetID() string {
	return c.id
}

type P2P struct {
	log          *logrus.Logger
	host         host.Host
	grpcServer   *grpc.Server
	PeerChan     chan peer.AddrInfo
	peerListChan chan peer.IDSlice
	clients      cmap.ConcurrentMap
	externalDB   p2psrv.ExternalDB
	prvKey       crypto.PrivKey
}

type P2PKey struct {
	prvKey crypto.PrivKey
}

func (p2p *P2PKey) Sign(commit string) (string, error) {
	sig, err := p2p.prvKey.Sign([]byte(commit))
	if err != nil {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	return base36.EncodeBytes(sig), nil
}

func (p2p *P2PKey) Verify(commit string, signature string, publicKey string) error {
	// Decode the base64-encoded public key string to bytes
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Unmarshal the public key bytes into a public key object
	pubKey, err := crypto.UnmarshalPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	// Decode the base64-encoded signature string to bytes
	signatureBytes := base36.DecodeToBytes(signature)

	// Verify the signature using the public key
	verified, err := pubKey.Verify([]byte(commit), signatureBytes)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !verified {
		return fmt.Errorf("verification failed for public key %s commit %s signature %s ", publicKey, commit, signature)
	}

	return nil
}

func (p2p *P2PKey) PublicKey() string {

	mPubKey, err := crypto.MarshalPublicKey(p2p.prvKey.GetPublic())
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(mPubKey)
}

func (p2p *P2PKey) PrivateKey() crypto.PrivKey {
	return p2p.prvKey
}

func (p2p *P2PKey) GetID() string {

	peerID, err := peer.IDFromPrivateKey(p2p.prvKey)
	if err != nil {
		panic(err)
	}

	return peerID.String()
}

func (p2p *P2P) HandlePeerFound(pi peer.AddrInfo) {
	p2p.PeerChan <- pi
}

func (p2p *P2P) GetClients() []*P2PClient {
	clients := []*P2PClient{}
	for _, c := range p2p.clients.Items() {
		clients = append(clients, c.(*P2PClient))
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
					if tries == 20 {
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
				client := &P2PClient{
					PingerClient: p2pproto.NewPingerClient(conn),
					TesterClient: p2pproto.NewTesterClient(conn),
					id:           peer.ID.String(),
				}

				// test connectivity with a ping
				_, err = client.Ping(ctx, &p2pproto.PingRequest{
					Ping: "pong",
				})
				if err != nil {
					p2p.log.Error("Ping failed: ", err)
					continue
				}

				p2p.log.Infof("Connected to %s", peer.ID.String())
				p2p.clients.Set(peer.ID.String(), client)
				if p2p.externalDB != nil {
					err = p2p.externalDB.AddPeer(peer.ID.String(), conn)
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
	if p2p.externalDB != nil {
		if err := p2p.externalDB.RemovePeer(conn.RemotePeer().String()); err != nil {
			p2p.log.Errorf("Failed to remove DB peer for '%s': %v", conn.RemotePeer().String(), err)
		}
	}
}

func (p2p *P2P) GetGRPCServer() *grpc.Server {
	return p2p.grpcServer
}

func (p2p *P2P) GetID() string {
	return p2p.host.ID().String()
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer() (func() error, error) {

	p2p.log.Infof("Starting p2p server using id %s", p2p.host.ID())
	ctx := context.TODO()

	// register internal grpc servers
	srv := &p2psrv.Server{DB: p2p.externalDB}
	p2pproto.RegisterPingerServer(p2p.grpcServer, srv)
	p2pproto.RegisterTesterServer(p2p.grpcServer, srv)

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

func NewKey(workdir string) (*P2PKey, error) {
	workdirInfo, err := os.Stat(workdir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(workdir, 0755)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		if !workdirInfo.IsDir() {
			return nil, fmt.Errorf("workdir %s is not a directory", workdir)
		}
	}

	var prvKey crypto.PrivKey
	keyFile := workdir + "/key"
	keyInfo, err := os.Stat(keyFile)
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
		if keyInfo.IsDir() {
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
	return &P2PKey{prvKey: prvKey}, nil
}

// NewManager creates and returns a new p2p manager
func NewManager(p2pkey *P2PKey, port int, peerListChan chan peer.IDSlice, logger *logrus.Logger, externalDB p2psrv.ExternalDB) (*P2P, error) {
	p2p := &P2P{
		PeerChan:     make(chan peer.AddrInfo),
		peerListChan: peerListChan,
		clients:      cmap.New(),
		log:          logger,
		grpcServer:   grpc.NewServer(p2pgrpc.WithP2PCredentials()),
		externalDB:   externalDB,
		prvKey:       p2pkey.PrivateKey(),
	}

	con, err := connmgr.NewConnManager(100, 400)
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		libp2p.Identity(p2p.prvKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", port),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(quic.NewTransport),
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
