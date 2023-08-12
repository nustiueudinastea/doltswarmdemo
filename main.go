package main

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protosio/distributeddolt/db"
	"github.com/protosio/distributeddolt/p2p"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var dbi *db.DB
var log = logrus.New()
var dbDir string
var commitListChan = make(chan []db.Commit, 100)
var peerListChan = make(chan peer.IDSlice, 1000)
var p2pmgr *p2p.P2P

type EventWriter struct {
	eventChan chan []byte
}

func (ew *EventWriter) Write(p []byte) (n int, err error) {
	logLine := make([]byte, len(p))
	copy(logLine, p)
	ew.eventChan <- logLine
	return len(logLine), nil
}

func p2pRun(dbDir string, port int) error {

	ew := &EventWriter{eventChan: make(chan []byte, 5000)}
	log.SetOutput(ew)

	p2pStopper, err := p2pmgr.StartServer()
	if err != nil {
		return err
	}

	dbi.StartUpdater()

	gui := createUI(peerListChan, commitListChan, ew.eventChan)
	// the following blocks so we can close everything else once this returns
	err = gui.Run()
	if err != nil {
		panic(err)
	}
	err = p2pStopper()
	if err != nil {
		return err
	}

	log.Info("Shutdown completed")

	return nil
}

func Init(localInit bool, peerInit string, port int) error {
	if localInit && peerInit != "" {
		return fmt.Errorf("cannot specify both local and peer init")
	}

	if localInit {
		return dbi.InitLocal()
	} else if peerInit != "" {
		var p2pStopper func() error
		var err error
		go func() {
			p2pStopper, err = p2pmgr.StartServer()
			if err != nil {
				panic(err)
			}
		}()

		err = dbi.InitFromPeer(peerInit)
		if err != nil {
			return fmt.Errorf("error initialising from peer: %w", err)
		}

		err = p2pStopper()
		if err != nil {
			return err
		}

		return nil
	} else {
		return fmt.Errorf("must specify either local or peer init")
	}
}

func main() {
	var port int
	var localInit bool
	var peerInit string

	funcBefore := func(ctx *cli.Context) error {
		var err error

		p2pmgr, err = p2p.NewManager(dbDir, port, peerListChan, log)
		if err != nil {
			return err
		}

		dbi, err = db.New(dbDir, commitListChan, p2pmgr, p2pmgr, log)
		if err != nil {
			log.Fatal(err)
		}
		err = dbi.Open()
		if err != nil {
			log.Fatal(err)
		}

		return nil
	}

	funcAfter := func(ctx *cli.Context) error {
		return dbi.Close()
	}

	app := &cli.App{
		Name: "distributeddolt",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "db",
				Value:       "db",
				Usage:       "db directory",
				Destination: &dbDir,
			},
			&cli.IntFlag{
				Name:        "port",
				Value:       10500,
				Usage:       "port number",
				Destination: &port,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Usage:  "starts p2p server",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return p2pRun(dbDir, port)
				},
			},
			{
				Name:  "init",
				Usage: "initialises db",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "local",
						Value:       false,
						Destination: &localInit,
					},
					&cli.StringFlag{
						Name:        "peer",
						Value:       "",
						Destination: &peerInit,
					},
				},
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return Init(localInit, peerInit, port)
				},
			},
			{
				Name:   "sql",
				Usage:  "runs SQL",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return dbi.Query(ctx.Args().First(), true)
				},
			},
			{
				Name:   "commits",
				Usage:  "list all commits",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return dbi.PrintAllCommits()
				},
			},
			{
				Name:   "data",
				Usage:  "show all data",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return dbi.PrintAllData()
				},
			},
			{
				Name:   "insert",
				Usage:  "insert data",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					return dbi.Insert(ctx.Args().First())
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

}
