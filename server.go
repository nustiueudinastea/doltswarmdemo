package main

import "github.com/libp2p/go-libp2p/core/peer"

// Server is a remote p2p server
type Server struct {
	p2p *P2P
}

// HandlerPing responds to a ping request on the server side
func (s *Server) HandlerPing(peer peer.ID, data interface{}) (interface{}, error) {
	return PingResp{}, nil
}

// HandlerEcho responds to an echo request on the server side
func (s *Server) HandlerEcho(peer peer.ID, data interface{}) error {
	log.Infof("Echo req from %s", peer.String())
	return nil
}
