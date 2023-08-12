package p2p

import (
	"context"
	"fmt"

	"github.com/protosio/distributeddolt/db/client"
	"github.com/protosio/distributeddolt/proto"
)

func (p2p *P2P) Ping() ([]*proto.PingResponse, error) {
	req := &proto.PingRequest{}

	responses := make([]*proto.PingResponse, 0)
	for clientIface := range p2p.clients.IterBuffered() {
		client, ok := clientIface.Val.(*P2PClient)
		if !ok {
			p2p.log.Errorf("Client %s has incorrect type", clientIface.Key)
			continue
		}

		resp, err := client.Ping(context.TODO(), req, nil)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func (p2p *P2P) AdvertiseHead(head string) error {

	req := &proto.AdvertiseHeadRequest{Head: head}

	responses := make([]*proto.AdvertiseHeadResponse, 0)
	for clientIface := range p2p.clients.IterBuffered() {
		client, ok := clientIface.Val.(*P2PClient)
		if !ok {
			p2p.log.Errorf("Client %s has incorrect type", clientIface.Key)
			continue
		}

		resp, err := client.AdvertiseHead(context.TODO(), req, nil)
		if err != nil {
			return err
		}
		responses = append(responses, resp)
	}
	p2p.log.Infof("Advertised head %s to peers: %v", head, responses)
	return nil
}

func (p2p *P2P) GetClient(id string) (client.Client, error) {
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
