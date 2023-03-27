package pinger

import (
	"context"
	"errors"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/protosio/testdolt/proto"
)

var _ proto.PingerServer = (*Server)(nil)

type Server struct{}

func (s *Server) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	_, ok := p2pgrpc.RemotePeerFromContext(ctx)
	if !ok {
		return nil, errors.New("no AuthInfo in context")
	}

	res := &proto.PingResponse{
		Pong: "Ping: " + req.Ping + "!",
	}
	return res, nil
}
