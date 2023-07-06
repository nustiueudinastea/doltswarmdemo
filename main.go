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

// func listBranches() {
// 	ddb, err := dbi.GetDoltDB()
// 	if err != nil {
// 		log.Fatalf(err.Error())
// 	}

// 	ctx := context.Background()
// 	headRefs, err := ddb.GetHeadRefs(ctx)
// 	if err != nil {
// 		log.Fatalf("failed to retrieve head refs: %s", err.Error())
// 	}
// 	fmt.Println(headRefs)
// }

type EventWriter struct {
	eventChan chan []byte
}

func (ew *EventWriter) Write(p []byte) (n int, err error) {
	logLine := make([]byte, len(p))
	copy(logLine, p)
	ew.eventChan <- logLine
	return len(logLine), nil
}

func localInit() error {
	err := dbi.Init()
	if err != nil {
		log.Fatal(err)
	}

	return err
}

// func p2pInit(dbDir string, port int) error {
// 	dbi = db.New(dbDir, log)
// 	err := dbi.OpenForInit()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	err = dbi.Query(fmt.Sprintf("CREATE DATABASE %s;", "test"), false)
// 	if err != nil {
// 		return fmt.Errorf("failed to create db: %w", err)
// 	}

// 	err = dbi.Query("show databases;", true)
// 	if err != nil {
// 		return fmt.Errorf("failed to create db: %w", err)
// 	}

// 	doltDB, err := dbi.GetDoltDB()
// 	if err != nil {
// 		return err
// 	}

// 	peerListChan := make(chan peer.IDSlice, 100)
// 	p2pmgr, err := p2p.NewManager(true, port, peerListChan, log, doltDB)
// 	if err != nil {
// 		return err
// 	}

// 	p2pStopper, err := p2pmgr.StartServer()
// 	if err != nil {
// 		return err
// 	}

// 	err = dbi.EnableP2P(p2pmgr)
// 	if err != nil {
// 		return err
// 	}

// 	err = dbi.InitP2P()
// 	if err != nil {
// 		return err
// 	}

// 	err = p2pStopper()
// 	if err != nil {
// 		return err
// 	}

// 	log.Info("Shutdown completed")

// 	return nil
// }

func p2pRun(dbDir string, port int) error {

	ew := &EventWriter{eventChan: make(chan []byte, 5000)}
	log.SetOutput(ew)

	peerListChan := make(chan peer.IDSlice, 1000)
	p2pmgr, err := p2p.NewManager(true, port, peerListChan, log, dbi)
	if err != nil {
		return err
	}

	p2pStopper, err := p2pmgr.StartServer()
	if err != nil {
		return err
	}

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

func main() {
	var port int
	var localinit bool

	funcBefore := func(ctx *cli.Context) error {
		dbi = db.New(dbDir, commitListChan, log)
		err := dbi.Open()
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
						Destination: &localinit,
					},
				},
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					if localinit {
						return localInit()
					}
					// } else {
					// 	return p2pInit(dbDir, port)
					// }
					return nil

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
