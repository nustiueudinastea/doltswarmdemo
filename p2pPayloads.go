package main

const (
	pingHandler               = "ping"
	pubsubEcho  pubsubMsgType = "echo"
)

type PingReq struct{}
type PingResp struct{}

type EchoReq struct{}
