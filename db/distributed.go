package db

import (
	"context"
	"fmt"

	"github.com/protosio/distributeddolt/db/server"
	"github.com/protosio/distributeddolt/proto"
	"google.golang.org/grpc"
)

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

//
// handlers
//

func (db *DB) remoteEventProcessor(broadcastEvents chan server.Event) func() error {
	db.log.Info("Starting db remote event processor")
	stopSignal := make(chan struct{})
	go func() {
		for {
			select {
			case event := <-broadcastEvents:
				err := db.eventHandler(event)
				if err != nil {
					db.log.Errorf("Error handling event '%s' from peer '%s': %v", event.Type, event.Peer, err)
				}
			case <-stopSignal:
				db.log.Info("Stopping db remote event processor")
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

func (db *DB) eventHandler(event server.Event) error {
	switch event.Type {
	case server.ExternalNewHeadEvent:
		db.log.Infof("new event '%s' from peer '%s': head -> %s", event.Type, event.Peer, event.Data.(string))
		err := db.Pull(event.Peer)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown event type '%s'", event.Type)
	}

	return nil
}

//
// client methods
//

func (db *DB) AdvertiseHead() {

	clients := db.clientRetriever.GetClients()

	if len(clients) == 0 {
		return
	}

	commit, err := db.GetLastCommit()
	if err != nil {
		db.log.Errorf("Error getting last commit: %v", err)
		return
	}

	req := &proto.AdvertiseHeadRequest{Head: commit.Hash}

	for _, client := range clients {
		_, err := client.AdvertiseHead(context.TODO(), req, grpc.WaitForReady(true))
		if err != nil {
			db.log.Errorf("Error advertising head to peer '%s': %v", client.GetID(), err)
			continue
		}

	}

	db.log.Infof("Advertised head %s to all peers", commit.Hash)
}