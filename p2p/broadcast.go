package p2p

import (
	"context"

	"github.com/protosio/distributeddolt/proto"
	"google.golang.org/grpc"
)

type BroadcastClient struct {
	p2p *P2P
}

func (bc *BroadcastClient) Ping(ctx context.Context, in *proto.PingRequest, opts ...grpc.CallOption) ([]*proto.PingResponse, error) {
	responses := make([]*proto.PingResponse, 0)
	for clientIface := range bc.p2p.clients.IterBuffered() {
		client, ok := clientIface.Val.(*Client)
		if !ok {
			bc.p2p.log.Errorf("Client %s has incorrect type", clientIface.Key)
			continue
		}

		resp, err := client.Ping(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}
