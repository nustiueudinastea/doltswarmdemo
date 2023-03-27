package main

// import (
// 	"fmt"
// 	"time"

// 	"github.com/libp2p/go-libp2p/core/peer"
// )

// type rpcClient struct {
// 	peer peer.ID
// 	p2p  *P2P
// }

// // Ping is a remote ping call to peer
// func (c *rpcClient) Ping() (time.Duration, error) {

// 	req := PingReq{}
// 	respData := &PingResp{}

// 	// send the request
// 	err := c.p2p.sendRequest(c.peer, pingHandler, req, respData)
// 	if err != nil {
// 		return 0, fmt.Errorf("ping request to '%s' failed: %s", c.peer.String(), err.Error())
// 	}

// 	return 0, nil
// }

// type PubSubClient struct {
// 	p2p *P2P
// }

// func (c *PubSubClient) BroadcastEcho() error {
// 	return c.p2p.BroadcastMsg(pubsubEcho, EchoReq{})
// }
