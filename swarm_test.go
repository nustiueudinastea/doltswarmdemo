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

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nustiueudinastea/doltswarm"
	swarmproto "github.com/nustiueudinastea/doltswarm/proto"
	"github.com/protosio/distributeddolt/p2p"
	p2pproto "github.com/protosio/distributeddolt/p2p/proto"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	gocmd "gopkg.in/ryankurte/go-async-cmd.v1"
)

//
// TODO:
// - test that commits are propagated
// - ability to wait for commits to be propagated (eventually with a timeout)
// - test conflic resolution using custom merge function
// - test denial of commit
//

const (
	localInit = "localInit"
	peerInit  = "peerInit"
	server    = "server"

	startPort = 10500
)

var logger = logrus.New()
var p2pStopper func() error

//
// testDB is a mock database
//

type testDB struct{}

func (pr *testDB) AddPeer(peerID string, conn *grpc.ClientConn) error {
	return nil
}
func (pr *testDB) RemovePeer(peerID string) error {
	return nil
}

func (pr *testDB) GetAllCommits() ([]doltswarm.Commit, error) {
	return []doltswarm.Commit{}, nil
}

func (pr *testDB) ExecAndCommit(query string, commitMsg string) (string, error) {
	return "", nil
}

func (pr *testDB) GetLastCommit(branch string) (doltswarm.Commit, error) {
	return doltswarm.Commit{}, nil
}

//
// ServerSyncer is a mock syncer
//

type ServerSyncer struct {
	peerCommits map[string][]string
}

func (s *ServerSyncer) AdvertiseHead(ctx context.Context, req *swarmproto.AdvertiseHeadRequest) (*swarmproto.AdvertiseHeadResponse, error) {
	peer, ok := p2pgrpc.RemotePeerFromContext(ctx)
	if !ok {
		return nil, errors.New("no AuthInfo in context")
	}
	if _, found := s.peerCommits[peer.String()]; found {
		s.peerCommits[peer.String()] = append(s.peerCommits[peer.String()], req.Head)
	} else {
		s.peerCommits[peer.String()] = []string{req.Head}
	}
	logger.Infof("AdvertiseHead from %s: %s", peer, req.Head)
	return &swarmproto.AdvertiseHeadResponse{}, nil
}

func (s *ServerSyncer) RequestHead(ctx context.Context, req *swarmproto.RequestHeadRequest) (*swarmproto.RequestHeadResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *ServerSyncer) peerHasCommit(peer string, commit string) bool {
	if commits, found := s.peerCommits[peer]; found {
		for _, c := range commits {
			if commit == c {
				return true
			}
		}
	}
	return false
}

//
// controller is a wrapper around a process
//

type controller struct {
	quitChan chan bool
	exitErr  chan error
	hostID   chan string
	name     string
}

func (c *controller) Name() string {
	return c.name
}

func runInstance(testDir string, nr int, mode string, initPeer string, printOutput bool) *controller {
	ctrl := &controller{
		quitChan: make(chan bool, 100),
		exitErr:  make(chan error, 100),
		hostID:   make(chan string, 100),
		name:     fmt.Sprintf("dsw%d", nr),
	}

	timeOutSeconds := 5
	port := 10500 + nr
	waitOutput := ""

	commands := []string{"--port", strconv.Itoa(port), "--db", testDir + "/" + ctrl.name}

	switch mode {
	case localInit:
		commands = append(commands, "init", "--local")
		waitOutput = "p2p setup done"
		timeOutSeconds = 10
	case peerInit:
		commands = append(commands, "init", "--peer", initPeer)
		waitOutput = "Successfully cloned db"
		timeOutSeconds = 30
	case server:
		commands = append(commands, "--no-gui", "--no-commits", "server")
		timeOutSeconds = 6000
	}

	go func() {
		timeout := time.After(time.Duration(timeOutSeconds) * time.Second)
		logger.Infof("starting proc '%s'(%s)", ctrl.name, mode)
		c := gocmd.Command("./ddolt", commands...)
		c.OutputChan = make(chan string, 1024)
		err := c.Start()
		if err != nil {
			ctrl.exitErr <- fmt.Errorf("Error starting proc '%s': %s\n", ctrl.name, err.Error())
			return
		}

		for {
			select {
			case line := <-c.OutputChan:
				if printOutput {
					fmt.Printf("stdout '%s': %s", ctrl.name, line)
				}

				if waitOutput != "" && strings.Contains(line, waitOutput) {
					if mode == peerInit {
						logger.Infof("%s: finished p2p clone", ctrl.name)
					} else {
						logger.Infof("%s: exiting", ctrl.name)
					}
					err := c.Exit()
					if err != nil {
						ctrl.exitErr <- fmt.Errorf("Error for proc '%s' when exiting: %s\n", ctrl.name, err.Error())
					}
				}

				if strings.Contains(line, "Shutdown completed") {
					ctrl.exitErr <- nil
				}

				if strings.Contains(line, "server using id") {
					tokens := strings.Split(line, " ")
					keyToken := tokens[7]
					hostID := keyToken[:len(keyToken)-2]
					ctrl.hostID <- hostID
				}
			case <-ctrl.quitChan:
				err := c.Exit()
				if err != nil {
					ctrl.exitErr <- fmt.Errorf("Error for proc '%s' when existing: %s\n", ctrl.name, err.Error())
				}
			case <-timeout:
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

func doInit(testDir string, nrOfInstances int) error {

	fmt.Println("==== Initialising test ====")

	ctrl1 := runInstance(testDir, 1, localInit, "", false)
	err := <-ctrl1.exitErr
	if err != nil {
		return fmt.Errorf("failed to local init instance: %s", err.Error())
	}

	ctrl1 = runInstance(testDir, 1, server, "", false)
	ctrl1Host := <-ctrl1.hostID
	defer func() {
		ctrl1.quitChan <- true
		err = <-ctrl1.exitErr
		if err != nil {
			logger.Error("failed to stop init instance: ", err)
		}
	}()

	clientInstances := make([]*controller, nrOfInstances-1)
	for i := 0; i < nrOfInstances-1; i++ {
		clientInstances[i] = runInstance(testDir, i+2, peerInit, ctrl1Host, false)
	}

	// print host ID for each instance
	logger.Infof("Instance '%s' has ID '%s'", ctrl1.Name(), ctrl1Host)
	for _, instance := range clientInstances {
		hostID := <-instance.hostID
		logger.Infof("Instance '%s' has ID '%s'", instance.Name(), hostID)
	}

	// wait for all instances to finish
	for _, instance := range clientInstances {
		instanceErr := <-instance.exitErr
		if instanceErr != nil {
			err = errors.Join(err, fmt.Errorf("instance '%s': %s", instance.Name(), instanceErr.Error()))
		}
	}
	if err != nil {
		return err
	}

	return nil
}

func stopAllInstances(instances []*controller, t *testing.T) {
	logger.Info("Stopping p2p instances")
	var err error
	for _, instance := range instances {
		instance.quitChan <- true
	}

	for _, instance := range instances {
		instanceErr := <-instance.exitErr
		if instanceErr != nil {
			err = errors.Join(err, fmt.Errorf("instance '%s': %s", instance.Name(), instanceErr.Error()))
		}
	}
	if err != nil {
		t.Fatal(err)
	}

	if p2pStopper != nil {
		err = p2pStopper()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestIntegration(t *testing.T) {

	testDir, err := os.MkdirTemp("temp", "tst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	nrOfInstances := 15
	err = doInit(testDir, nrOfInstances)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("==== Starting test %s ====\n", testDir)

	instances := make([]*controller, nrOfInstances)
	for i := 1; i <= nrOfInstances; i++ {
		instances[i-1] = runInstance(testDir, i, server, "", true)
	}
	defer stopAllInstances(instances, t)

	instanceIDs := make(map[string]string, nrOfInstances)
	// wait for all host ids
	for _, instance := range instances {
		hostID := <-instance.hostID
		instanceIDs[hostID] = instance.Name()
	}

	time.Sleep(10 * time.Second)

	peerListChan := make(chan peer.IDSlice, 100)
	tDB := &testDB{}
	p2pmgr, err = p2p.NewManager(testDir+"/testp2p", startPort, peerListChan, logger, tDB)
	if err != nil {
		t.Fatal(err)
	}

	grpcServer := p2pmgr.GetGRPCServer()
	srvSyncer := &ServerSyncer{peerCommits: make(map[string][]string)}
	swarmproto.RegisterDBSyncerServer(grpcServer, srvSyncer)

	p2pStopper, err = p2pmgr.StartServer()
	if err != nil {
		t.Fatal(err)
	}

	for len(p2pmgr.GetClients()) != nrOfInstances {
		logger.Infof("Waiting for %d clients. Currently have %d", nrOfInstances, len(p2pmgr.GetClients()))
		time.Sleep(2 * time.Second)
	}

	clients := p2pmgr.GetClients()
	for _, client := range clients {
		_, err = client.Ping(context.Background(), &p2pproto.PingRequest{
			Ping: "pong",
		})
		if err != nil {
			t.Fatalf("failure to ping client '%s': %s", client.GetID(), err.Error())
		}
	}

	logger.Infof("test p2p ID: %s", p2pmgr.GetID())

	//
	// Check that all clients have the same head
	//
	allHeads := make(map[string]string)
	for _, client := range clients {
		resp, instanceErr := client.GetHead(context.Background(), &p2pproto.GetHeadRequest{})
		if instanceErr != nil {
			err = errors.Join(err, fmt.Errorf("failure while calling GetHead on instance '%s'(%s): %s", instanceIDs[client.GetID()], client.GetID(), instanceErr.Error()))
			continue
		}
		allHeads[client.GetID()] = resp.Commit
	}
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(allHeads) != nrOfInstances {
		t.Error("not all instances have a head. Expected: ", nrOfInstances, " Got: ", len(allHeads))
	}
	for _, head := range allHeads {
		if head != allHeads[clients[0].GetID()] {
			t.Errorf("heads are not the same: %v", allHeads)
		}
	}

	//
	// Insert and make sure commit is propagated
	//
	uid, err := ksuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	queryString := fmt.Sprintf("INSERT INTO %s (id, name) VALUES ('%s', '%s');", tableName, uid.String(), "propagation test")
	resp, err := clients[0].ExecSQL(context.Background(), &p2pproto.ExecSQLRequest{Statement: queryString, Msg: "commit propagation test"})
	if err != nil {
		t.Fatal(err)
	}

	passed := false
	timeStart := time.Now()
	for i := 0; i < 200; i++ {
		for _, client := range clients {
			if !srvSyncer.peerHasCommit(client.GetID(), resp.Commit) {
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}
		passed = true
		break
	}
	if !passed {
		t.Fatal("commit not propagated to all peers")
	} else {
		logger.Infof("It took %f seconds to propagate to %d clients", time.Since(timeStart).Seconds(), len(clients))
	}

}
