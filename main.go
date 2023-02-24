package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var db = DB{}
var log = logrus.New()
var dbDir string

// func createBranch(name string) error {
// 	ddb, err := db.GetDoltDB()
// 	if err != nil {
// 		return err
// 	}

// 	dref, err := ref.Parse(fmt.Sprintf("refs/heads/%s", name))
// 	if err != nil {
// 		return err
// 	}

// 	commit, err := db.GetLastCommit()
// 	if err != nil {
// 		return err
// 	}

// 	ctx := context.Background()
// 	commitVal, err := ddb.ReadCommit(ctx, hash.Parse(commit.Hash))
// 	if err != nil {
// 		log.Fatalf("failed to retrieve commit value: %s", err.Error())
// 	}

// 	return ddb.NewBranchAtCommit(ctx, dref, commitVal)
// }

// func showData(branch string) {
// 	db.PrintQuery(fmt.Sprintf("SELECT * FROM `%s/%s`.protos;", dbName, branch))
// }

func listCommits(branch string) {
	query(fmt.Sprintf("select * from `%s/%s`.dolt_diff;", dbName, branch))
}

func listBranches() {
	ddb, err := db.GetDoltDB()
	if err != nil {
		log.Fatalf(err.Error())
	}

	ctx := context.Background()
	headRefs, err := ddb.GetHeadRefs(ctx)
	if err != nil {
		log.Fatalf("failed to retrieve head refs: %s", err.Error())
	}
	fmt.Println(headRefs)
}

func query(query string) {
	commitListChan := make(chan []Commit, 100)
	err := db.Open(dbDir, commitListChan)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.PrintQuery(query)
}

// func insert(branch string, data string) error {
// 	return db.Insert(branch, data)
// }

type EventWriter struct {
	eventChan chan []byte
}

func (ew *EventWriter) Write(p []byte) (n int, err error) {
	ew.eventChan <- p
	return len(p), nil
}

func p2p(dbDir string, port int) error {

	ew := &EventWriter{eventChan: make(chan []byte, 500)}
	log.SetOutput(ew)

	commitListChan := make(chan []Commit, 100)
	err := db.Open(dbDir, commitListChan)
	if err != nil {
		return err
	}
	defer db.Close()

	// listBranches()

	// listCommits("main")

	peerListChan := make(chan peer.IDSlice, 100)
	p2p, err := NewManager(true, port, peerListChan)
	if err != nil {
		return err
	}

	p2pStopper, err := p2p.StartServer()
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

	app := &cli.App{
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
				Name:  "server",
				Usage: "starts p2p server",
				Action: func(ctx *cli.Context) error {
					return p2p(dbDir, port)
				},
			},
			{
				Name:  "sql",
				Usage: "runs SQL",
				Action: func(ctx *cli.Context) error {
					query(ctx.Args().First())
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}

	// ddb, err := db.GetDoltDB()
	// if err != nil {
	// 	log.Fatalf("failed to retrieve dolt db: %s", err.Error())
	// }

	// commitVal, err := ddb.ReadCommit(ctx, hash.Parse(commit.Hash))
	// if err != nil {
	// 	log.Fatalf("failed to retrieve commit value: %s", err.Error())
	// }

	// hash, err := commitVal.HashOf()
	// if err != nil {
	// 	log.Fatalf("failed to retrieve commit hash: %s", err.Error())
	// }

	// cm, err := commitVal.GetCommitMeta(ctx)
	// if err != nil {
	// 	log.Fatalf("failed to retrieve commit meta: %s", err.Error())
	// }

	// err = ddb.SetHead(ctx, headRefs[0], hash)
	// if err != nil {
	// 	log.Fatalf("failed to set head at commit: %s", err.Error())
	// }

	// ddb.CommitWithWorkingSet()

	// ddb.CommitRoot()

	// _, err = ddb.NewPendingCommit()
	// if err != nil {
	// 	log.Fatalf("failed to create new commit: %s", err.Error())
	// }

}
