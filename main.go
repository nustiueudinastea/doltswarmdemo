package main

import (
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nustiueudinastea/doltswarm"
	"github.com/protosio/distributeddolt/p2p"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var dbi *doltswarm.DB
var log = logrus.New()
var workDir string
var commitListChan = make(chan []doltswarm.Commit, 100)
var peerListChan = make(chan peer.IDSlice, 1000)
var p2pmgr *p2p.P2P
var uiLog = &EventWriter{eventChan: make(chan []byte, 5000)}

type EventWriter struct {
	eventChan chan []byte
}

func (ew *EventWriter) Write(p []byte) (n int, err error) {
	logLine := make([]byte, len(p))
	copy(logLine, p)
	ew.eventChan <- logLine
	return len(logLine), nil
}

func p2pRun(workDir string, port int) error {

	p2pStopper, err := p2pmgr.StartServer()
	if err != nil {
		return err
	}

	updaterSopper := startCommitUpdater()

	gui := createUI(peerListChan, commitListChan, uiLog.eventChan)
	// the following blocks so we can close everything else once this returns
	err = gui.Run()
	if err != nil {
		panic(err)
	}

	// time.Sleep(time.Second * 300)

	err = p2pStopper()
	if err != nil {
		log.Error(err)
	}

	err = updaterSopper()
	if err != nil {
		log.Error(err)
	}

	log.Info("Shutdown completed")

	return nil
}

func startCommitUpdater() func() error {
	log.Info("Starting commit updater")
	updateTimer := time.NewTicker(1 * time.Second)
	commitTimmer := time.NewTicker(15 * time.Second)
	stopSignal := make(chan struct{})
	go func() {
		for {
			select {
			case <-updateTimer.C:
				commits, err := dbi.GetAllCommits()
				if err != nil {
					log.Errorf("failed to retrieve all commits: %s", err.Error())
					continue
				}
				commitListChan <- commits
			case timer := <-commitTimmer.C:
				err := insert(timer.String())
				if err != nil {
					log.Errorf("Failed to insert time: %s", err.Error())
					continue
				}
				log.Infof("Inserted time '%s' into db", timer.String())
			case <-stopSignal:
				log.Info("Stopping commit updater")
				return
			}
		}
	}()
	stopper := func() error {
		stopSignal <- struct{}{}
		return nil
	}
	return stopper
}

func Init(localInit bool, peerInit string, port int) error {
	if localInit && peerInit != "" {
		return fmt.Errorf("cannot specify both local and peer init")
	}

	if localInit {
		err := dbi.InitLocal()
		if err != nil {
			return fmt.Errorf("failed to init local db: %w", err)
		}

		tx, err := dbi.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// create table
		_, err = tx.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s(
			  id varchar(256) PRIMARY KEY,
			  name varchar(512)
			);`, tableName))
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		// add
		_, err = tx.Exec(`CALL DOLT_ADD('-A');`)
		if err != nil {
			return fmt.Errorf("failed to commit table: %w", err)
		}

		// commit
		_, err = tx.Exec(`CALL DOLT_COMMIT('-m', 'Initialize doltswarmdemo', '--author', 'Alex Giurgiu <alex@giurgiu.io>');`)
		if err != nil {
			return fmt.Errorf("failed to commit table: %w", err)
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
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
	var logLevel string

	funcBefore := func(ctx *cli.Context) error {
		var err error

		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %v", err)
		}

		log.SetLevel(level)

		if ctx.Command.Name != "init" {
			log.SetOutput(uiLog)
		}

		err = ensureDir(workDir)
		if err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}

		dbi, err = doltswarm.New(workDir, "doltswarmdemo", log)
		if err != nil {
			return fmt.Errorf("failed to create db: %v", err)
		}

		p2pmgr, err = p2p.NewManager(workDir, port, peerListChan, log, dbi)
		if err != nil {
			return fmt.Errorf("failed to create p2p manager: %v", err)
		}

		// grpc server needs to be added before opening the DB
		dbi.AddGRPCServer(p2pmgr.GetGRPCServer())

		err = dbi.Open()
		if err != nil {
			return fmt.Errorf("failed to open db: %v", err)
		}

		return nil
	}

	funcAfter := func(ctx *cli.Context) error {
		if dbi != nil {
			return dbi.Close()
		}
		return nil
	}

	app := &cli.App{
		Name: "distributeddolt",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log",
				Value:       "info",
				Usage:       "logging level",
				Destination: &logLevel,
			},
			&cli.StringFlag{
				Name:        "db",
				Value:       "db",
				Usage:       "db directory",
				Destination: &workDir,
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
					return p2pRun(workDir, port)
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
				Name:   "status",
				Usage:  "status info",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					fmt.Printf("PEER ID: %s\n", p2pmgr.GetID())
					return nil
				},
			},
			{
				Name:   "sql",
				Usage:  "runs SQL",
				Before: funcBefore,
				After:  funcAfter,
				Action: func(ctx *cli.Context) error {
					_, err := dbi.Query(ctx.Args().First())
					return err
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
					return insert(ctx.Args().First())
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

}
