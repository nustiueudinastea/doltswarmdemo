package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/dolthub/dolt/go/cmd/dolt/commands/engine"
	"github.com/dolthub/dolt/go/libraries/doltcore/dbfactory"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/env/actions"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/binlogreplication"
	"github.com/dolthub/dolt/go/libraries/utils/filesys"
	"github.com/dolthub/dolt/go/store/chunks"
	"github.com/dolthub/dolt/go/store/datas"
	"github.com/dolthub/dolt/go/store/types"
	dd "github.com/dolthub/driver"
	doltSQL "github.com/dolthub/go-mysql-server/sql"
	"github.com/protosio/distributeddolt/dbclient"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
)

type Commit struct {
	Hash         string
	Table        string
	Committer    string
	Email        string
	Date         time.Time
	Message      string
	DataChange   bool
	SchemaChange bool
}

var dbName = "protos"
var tableName = "testtable"

type DB struct {
	stopper        func() error
	commitListChan chan []Commit
	dbEnvInit      *env.DoltEnv
	mrEnv          *env.MultiRepoEnv
	sqle           *engine.SqlEngine
	sqld           *sql.DB
	sqlCtx         *doltSQL.Context
	workingDir     string
	log            *logrus.Logger
}

func New(dir string, commitListChan chan []Commit, logger *logrus.Logger) (*DB, error) {
	workingDir, err := filesys.LocalFS.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %v", workingDir, err)
	}

	db := &DB{
		workingDir:     workingDir,
		log:            logger,
		commitListChan: commitListChan,
	}

	return db, nil
}

func (db *DB) Open() error {

	workingDirFS, err := filesys.LocalFS.WithWorkingDir(db.workingDir)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	err = ensureDir(db.workingDir)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	ctx := context.Background()
	dbEnv := env.Load(ctx, env.GetCurrentUserHomeDir, workingDirFS, "file://"+db.workingDir+"/"+dbName, "1.6.1")
	db.dbEnvInit = dbEnv
	err = dbEnv.Config.WriteableConfig().SetStrings(map[string]string{
		env.UserEmailKey: "alex@giurgiu.io",
		env.UserNameKey:  "Alex Giurgiu",
	})
	if err != nil {
		return fmt.Errorf("failed to set config : %w", err)
	}

	db.mrEnv, err = env.MultiEnvForDirectory(ctx, dbEnv.Config.WriteableConfig(), workingDirFS, dbEnv.Version, dbEnv.IgnoreLockFile, dbEnv)
	if err != nil {
		return fmt.Errorf("failed to load database names: %v", err)
	}

	sqleConfig := &engine.SqlEngineConfig{
		IsReadOnly:              false,
		PrivFilePath:            ".doltcfg/privileges.db",
		BranchCtrlFilePath:      ".doltcfg/branch_control.db",
		DoltCfgDirPath:          ".doltcfg",
		ServerUser:              "root",
		ServerPass:              "",
		ServerHost:              "localhost",
		Autocommit:              true,
		DoltTransactionCommit:   true,
		JwksConfig:              []engine.JwksConfig{},
		ClusterController:       nil,
		BinlogReplicaController: binlogreplication.DoltBinlogReplicaController,
	}

	db.sqle, err = engine.NewSqlEngine(ctx, db.mrEnv, sqleConfig)
	if err != nil {
		return fmt.Errorf("failed to create sql engine: %w", err)
	}

	db.sqlCtx, err = db.sqle.NewLocalContext(ctx)
	if err != nil {
		return err
	}

	db.sqld = sql.OpenDB(&Connector{driver: &doltDriver{conn: &dd.DoltConn{DataSource: &dd.DoltDataSource{}, SE: db.sqle, GmsCtx: db.sqlCtx}}})

	return nil
}

func (db *DB) Close() error {

	if db.mrEnv != nil {
		dbEnv := db.mrEnv.GetEnv(dbName)
		if dbEnv != nil {
			remotes, err := dbEnv.GetRemotes()
			if err == nil {
				for r := range remotes {
					err = dbEnv.RemoveRemote(context.TODO(), r)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	err := db.sqle.Close()
	if err != context.Canceled {
		return err
	}

	if db.stopper != nil {
		db.stopper()
	}
	return nil
}

func (db *DB) InitLocal() error {
	err := db.Query(fmt.Sprintf("CREATE DATABASE %s;", dbName), true)
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}

	err = db.Query(fmt.Sprintf("USE %s;", dbName), false)
	if err != nil {
		return fmt.Errorf("failed to use db: %w", err)
	}

	// create table
	err = db.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s(
		  id varchar(256) PRIMARY KEY,
		  name varchar(512)
	    );`, tableName), true)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

func (db *DB) StartUpdater() {
	db.stopper = db.commitUpdater()
}

func (db *DB) GetFilePath() string {
	return db.workingDir + "/" + dbName
}

func (db *DB) GetChunkStore() (chunks.ChunkStore, error) {
	env := db.mrEnv.GetEnv(dbName)
	if env == nil {
		return nil, fmt.Errorf("failed to retrieve db env")
	}

	dbd := doltdb.HackDatasDatabaseFromDoltDB(env.DoltDB)
	return datas.ChunkStoreFromDatabase(dbd), nil
}

func (db *DB) EnableSync(cr dbclient.ClientRetriever) error {
	db.log.Info("Enabling p2p sync")
	dbfactory.RegisterFactory("protos", dbclient.NewCustomFactory(cr))
	return nil
}

func (db *DB) AddRemote(peerID string) error {

	db.log.Infof("Adding remote for peer %s", peerID)
	r := env.NewRemote(peerID, fmt.Sprintf("protos://%s", peerID), map[string]string{})
	dbEnv := db.mrEnv.GetEnv(dbName)
	if dbEnv == nil {
		dbEnv = db.dbEnvInit
		ddb, err := r.GetRemoteDB(context.TODO(), types.Format_Default, dbEnv)
		if err != nil {
			return fmt.Errorf("failed to get remote db: %v", err)
		}

		workingDir, err := filesys.LocalFS.Abs(db.workingDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %v", workingDir, err)
		}

		dbEnv, err = actions.EnvForClone(context.TODO(), ddb.ValueReadWriter().Format(), r, workingDir+"/"+dbName, db.dbEnvInit.FS, db.dbEnvInit.Version, env.GetCurrentUserHomeDir)
		if err != nil {
			return fmt.Errorf("failed to create clone env: %v", err)
		}

		workingDirFS, err := filesys.LocalFS.WithWorkingDir(workingDir)
		if err != nil {
			return fmt.Errorf("failed to open db: %w", err)
		}

		db.mrEnv, err = env.MultiEnvForDirectory(context.TODO(), dbEnv.Config.WriteableConfig(), workingDirFS, dbEnv.Version, dbEnv.IgnoreLockFile, dbEnv)
		if err != nil {
			return fmt.Errorf("failed to create mr env: %v", err)
		}
	} else {
		remotes, err := dbEnv.GetRemotes()
		if err != nil {
			return fmt.Errorf("failed to get remotes: %v", err)
		}
		if _, ok := remotes[peerID]; !ok {
			err := dbEnv.AddRemote(r)
			if err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}
		} else {
			db.log.Infof("Remote for peer %s already exists", peerID)
		}
	}

	return nil
}

func (db *DB) InitFromPeer() error {
	db.log.Info("Initializing from first peer")
	var peerID string
	for {
		dbEnv := db.mrEnv.GetEnv(dbName)
		if dbEnv == nil {
			db.log.Info("init from peer: db env is nil")
			time.Sleep(2 * time.Second)
			continue
		}
		remotes, err := dbEnv.GetRemotes()
		if err != nil {
			db.log.Warnf("init from peer: failed to get remotes: %v", err)
		}
		if len(remotes) == 0 {
			db.log.Info("waiting for at least one peer to be added")
			time.Sleep(2 * time.Second)
			continue
		} else {
			for k := range remotes {
				peerID = k
				break
			}
			break
		}
	}

	db.log.Infof("initializing from peer %s", peerID)

	err := db.Query(fmt.Sprintf("CALL DOLT_CLONE('protos://%s', 'main');", peerID), true)
	if err != nil {
		return fmt.Errorf("failed to clone db: %w", err)
	}

	// dbEnv := db.mrEnv.GetEnv(dbName)
	// if dbEnv == nil {
	// 	return fmt.Errorf("init from peer: db env is nil")
	// }
	// remotes, err := dbEnv.GetRemotes()
	// if err != nil {
	// 	return fmt.Errorf("init from peer: failed to get remotes: %v", err)
	// }

	// r := remotes[peerID]

	// ddb, err := r.GetRemoteDB(context.Background(), types.Format_Default, dbEnv)
	// if err != nil {
	// 	return fmt.Errorf("init from peer: failed to get remote db: %v", err)
	// }

	// err = actions.CloneRemote(context.Background(), ddb, peerID, "main", dbEnv)
	// if err != nil {
	// 	return fmt.Errorf("failed to clone db: %w", err)
	// }

	return nil
}

func (db *DB) RemoveRemote(peerID string) error {
	dbEnv := db.mrEnv.GetEnv(dbName)
	if dbEnv == nil {
		return nil
	}

	err := dbEnv.RemoveRemote(context.Background(), peerID)
	if err != nil {
		return fmt.Errorf("failed to remove remote: %w", err)
	}

	return nil
}

func (db *DB) Sync(peerID string) error {
	err := db.Query(fmt.Sprintf("USE %s;", dbName), false)
	if err != nil {
		return fmt.Errorf("failed to use db: %w", err)
	}

	err = db.Query("CALL DOLT_PULL('origin', 'main');", true)
	if err != nil {
		return fmt.Errorf("failed to sync db: %w", err)
	}
	return nil
}

func (db *DB) Query(query string, printResult bool) (err error) {
	schema, rows, err := db.sqle.Query(db.sqlCtx, query)
	if err != nil {
		return err
	}

	if printResult {
		engine.PrettyPrintResults(db.sqlCtx, engine.FormatTabular, schema, rows)
	} else {
		for {
			_, err := rows.Next(db.sqlCtx)
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
		}
		rows.Close(db.sqlCtx)
	}

	return nil
}

func (db *DB) Insert(data string) error {
	uid, err := ksuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to create uid: %w", err)
	}

	err = db.Query(fmt.Sprintf("USE %s;", dbName), false)
	if err != nil {
		return fmt.Errorf("failed to use db: %w", err)
	}

	queryString := fmt.Sprintf("INSERT INTO %s (id, name) VALUES ('%s', '%s');", tableName, uid.String(), data)
	err = db.Query(queryString, false)
	if err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	return nil
}

func (db *DB) GetLastCommit() (Commit, error) {

	query := fmt.Sprintf("SELECT {*} FROM `%s/main`.dolt_diff ORDER BY date DESC;", dbName)
	commits, err := sq.FetchAll(db.sqld, sq.
		Queryf(query).
		SetDialect(sq.DialectMySQL),
		commitMapper,
	)
	if err != nil {
		return Commit{}, fmt.Errorf("failed to retrieve last commit hash: %w", err)
	}

	if len(commits) == 0 {
		return Commit{}, fmt.Errorf("no commits found")
	}

	return commits[0], nil
}

func (db *DB) GetAllCommits() ([]Commit, error) {
	query := fmt.Sprintf("SELECT {*} FROM `%s/main`.dolt_diff ORDER BY date DESC;", dbName)
	commits, err := sq.FetchAll(db.sqld, sq.
		Queryf(query).
		SetDialect(sq.DialectMySQL),
		commitMapper,
	)
	if err != nil {
		return commits, fmt.Errorf("failed to retrieve last commit hash: %w", err)
	}

	return commits, nil
}

func (db *DB) PrintAllCommits() error {
	query := fmt.Sprintf("SELECT * FROM `%s/main`.dolt_diff ORDER BY date;", dbName)
	err := db.Query(query, true)
	if err != nil {
		return fmt.Errorf("failed to retrieve commits: %w", err)
	}

	return nil
}

func (db *DB) PrintAllData() error {
	err := db.Query(fmt.Sprintf("SELECT * FROM `%s/main`.%s;", dbName, tableName), true)
	if err != nil {
		return fmt.Errorf("failed to retrieve commits: %w", err)
	}

	return nil
}

func (db *DB) PrintBranches() error {
	dbEnv := db.mrEnv.GetEnv(dbName)
	if dbEnv == nil {
		return fmt.Errorf("db '%s' not found", dbName)
	}

	ctx := context.Background()
	headRefs, err := dbEnv.DoltDB.GetHeadRefs(ctx)
	if err != nil {
		log.Fatalf("failed to retrieve head refs: %s", err.Error())
	}
	fmt.Println(headRefs)
	return nil
}

func (db *DB) commitUpdater() func() error {
	updateTimer := time.NewTicker(1 * time.Second)
	commitTimmer := time.NewTicker(120 * time.Second)
	stopSignal := make(chan struct{})
	go func() {
		for {
			select {
			case <-updateTimer.C:
				commits, err := db.GetAllCommits()
				if err != nil {
					db.log.Errorf("failed to retrieve all commits: %s", err.Error())
					continue
				}
				db.commitListChan <- commits
			case timer := <-commitTimmer.C:
				err := db.Insert(timer.String())
				if err != nil {
					db.log.Errorf("Failed to insert time: %s", err.Error())
					continue
				}
				db.log.Infof("Inserted time '%s' into db", timer.String())
			case <-stopSignal:
				db.log.Info("Stopping commit updater")
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
