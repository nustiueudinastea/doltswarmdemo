package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protosio/distributeddolt/p2p"
	p2pproto "github.com/protosio/distributeddolt/p2p/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	gocmd "gopkg.in/ryankurte/go-async-cmd.v1"
)

const (
	localInit = iota
	peerInit
	server

	startPort = 10500
)

var logger = logrus.New()
var p2pStopper func() error

type testPeerRegistrator struct{}

func (pr *testPeerRegistrator) AddPeer(peerID string, conn *grpc.ClientConn) error {
	return nil
}
func (pr *testPeerRegistrator) RemovePeer(peerID string) error {
	return nil
}

type controller struct {
	quitChan chan bool
	exitErr  chan error
	hostID   chan string
}

func runInstance(testDir string, nr int, mode int, initPeer string) *controller {
	ctrl := &controller{
		quitChan: make(chan bool, 100),
		exitErr:  make(chan error, 100),
		hostID:   make(chan string, 100),
	}

	timeOutSeconds := 5
	port := 10500 + nr
	name := fmt.Sprintf("dsw%d", nr)
	waitOutput := ""

	commands := []string{}

	switch mode {
	case localInit:
		commands = []string{"--port", strconv.Itoa(port), "--db", testDir + "/" + name, "init", "--local"}
		waitOutput = "p2p setup done"
		timeOutSeconds = 10
	case peerInit:
		commands = []string{"--port", strconv.Itoa(port), "--db", testDir + "/" + name, "init", "--peer", initPeer}
		waitOutput = "Successfully cloned db"
		timeOutSeconds = 30
	case server:
		commands = []string{"--port", strconv.Itoa(port), "--no-gui", "--db", testDir + "/" + name, "--no-commits", "server"}
		timeOutSeconds = 6000
	}

	go func() {
		timeout := time.After(time.Duration(timeOutSeconds) * time.Second)
		fmt.Printf("starting proc '%s'\n", name)
		c := gocmd.Command("./ddolt", commands...)
		c.OutputChan = make(chan string, 1024)
		err := c.Start()
		if err != nil {
			ctrl.exitErr <- fmt.Errorf("Error starting proc '%s': %s\n", name, err.Error())
			return
		}

		for {
			select {
			case line := <-c.OutputChan:
				fmt.Printf("stdout '%s': %s", name, line)
				if waitOutput != "" && strings.Contains(line, waitOutput) {
					err := c.Exit()
					if err != nil {
						ctrl.exitErr <- fmt.Errorf("Error for proc '%s' when exiting: %s\n", name, err.Error())
					} else {
						ctrl.exitErr <- nil
					}
					return
				}

				if strings.Contains(line, "server using id") {
					tokens := strings.Split(line, " ")
					keyToken := tokens[7]
					ctrl.hostID <- keyToken[:len(keyToken)-2]
				}
			case <-ctrl.quitChan:
				err := c.Exit()
				if err != nil {
					ctrl.exitErr <- fmt.Errorf("Error for proc '%s' when existing: %s\n", name, err.Error())
				} else {
					ctrl.exitErr <- nil
				}
				return
			case <-timeout:
				fmt.Printf("timeout waiting for '%s': %s\n", name, waitOutput)
				err := c.Exit()
				if err != nil {
					ctrl.exitErr <- fmt.Errorf("Timeout waiting for output '%s': %s", waitOutput, err.Error())
				} else {
					ctrl.exitErr <- fmt.Errorf("Timeout waiting for output '%s'", waitOutput)
				}
				return
			}
		}
	}()

	return ctrl
}

func doInit(testDir string) error {

	fmt.Println("==== Initialising test ====")

	ctrl1 := runInstance(testDir, 1, localInit, "")
	err := <-ctrl1.exitErr
	if err != nil {
		return err
	}
	defer func() {
		ctrl1.quitChan <- true
	}()

	ctrl1 = runInstance(testDir, 1, server, "")
	ctrl1Host := <-ctrl1.hostID

	ctrl2 := runInstance(testDir, 2, peerInit, ctrl1Host)
	ctrl3 := runInstance(testDir, 3, peerInit, ctrl1Host)
	ctrl4 := runInstance(testDir, 4, peerInit, ctrl1Host)
	ctrl5 := runInstance(testDir, 5, peerInit, ctrl1Host)
	ctrl6 := runInstance(testDir, 6, peerInit, ctrl1Host)

	err2 := <-ctrl2.exitErr
	err3 := <-ctrl3.exitErr
	err4 := <-ctrl4.exitErr
	err5 := <-ctrl5.exitErr
	err6 := <-ctrl6.exitErr

	err = errors.Join(err2, err3, err4, err5, err6)
	if err != nil {
		return err
	}

	ctrl1.quitChan <- true
	err = <-ctrl1.exitErr
	if err != nil {
		return err
	}

	return nil
}

func TestIntegration(t *testing.T) {

	testDir, err := os.MkdirTemp("temp", "tst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	err = doInit(testDir)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("==== Starting test ====")

	ctrl1 := runInstance(testDir, 1, server, "")
	ctrl2 := runInstance(testDir, 2, server, "")
	ctrl3 := runInstance(testDir, 3, server, "")
	ctrl4 := runInstance(testDir, 4, server, "")
	ctrl5 := runInstance(testDir, 5, server, "")
	ctrl6 := runInstance(testDir, 6, server, "")

	defer func() {

		logger.Info("Stopping p2p instances")

		ctrl1.quitChan <- true
		ctrl2.quitChan <- true
		ctrl3.quitChan <- true
		ctrl4.quitChan <- true
		ctrl5.quitChan <- true
		ctrl6.quitChan <- true

		err1 := <-ctrl1.exitErr
		err2 := <-ctrl2.exitErr
		err3 := <-ctrl3.exitErr
		err4 := <-ctrl4.exitErr
		err5 := <-ctrl5.exitErr
		err6 := <-ctrl6.exitErr

		err = errors.Join(err1, err2, err3, err4, err5, err6)
		if err != nil {
			t.Fatal(err)
		}

		if p2pStopper != nil {
			err = p2pStopper()
			if err != nil {
				t.Fatal(err)
			}
		}

	}()

	time.Sleep(15 * time.Second)

	peerListChan := make(chan peer.IDSlice, 100)
	peerRegistrator := &testPeerRegistrator{}
	p2pMgr, err := p2p.NewManager(testDir+"/testp2p", startPort, peerListChan, logger, peerRegistrator)
	if err != nil {
		t.Fatal(err)
	}

	p2pStopper, err = p2pMgr.StartServer()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Second)

	clients := p2pMgr.GetClients()
	for _, client := range clients {
		logger.Infof("Pinging client '%s'\n", client.GetID())
		_, err = client.Ping(context.Background(), &p2pproto.PingRequest{
			Ping: "pong",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
