package db

import "google.golang.org/grpc"

type PeerHandler interface {
	AddPeer(peerID string) error
	RemovePeer(peerID string) error
}

type PeerHandlerRegistrator interface {
	RegisterPeerHandler(handler PeerHandler)
}

type GRPCServerRetriever interface {
	GetGRPCServer() *grpc.Server
}
