package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/dolthub/dolt/go/cmd/dolt/commands/engine"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/binlogreplication"
	"github.com/dolthub/dolt/go/libraries/utils/filesys"
	dd "github.com/dolthub/driver"
	doltSQL "github.com/dolthub/go-mysql-server/sql"
	"github.com/protosio/distributeddolt/p2p"
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
	p2p            *p2p.P2P
	mrEnv          *env.MultiRepoEnv
	sqle           *engine.SqlEngine
	sqld           *sql.DB
	sqlCtx         *doltSQL.Context
	workingDir     string
	log            *logrus.Logger
}

func New(dir string, commitListChan chan []Commit, logger *logrus.Logger) *DB {
	db := &DB{
		workingDir:     dir,
		log:            logger,
		commitListChan: commitListChan,
	}

	return db
}

// func (db *DB) InitP2P() error {
// 	if db.sqle == nil {
// 		return errors.New("db not opened")
// 	}

// 	err := db.Query(fmt.Sprintf("CALL dolt_clone('-remote', 'origin', 'protos://%s', '%s' );", dbName, dbName), false)
// 	if err != nil {
// 		return fmt.Errorf("failed to clone db: %w", err)
// 	}
// 	return nil
// }

// func (db *DB) OpenForInit() error {
// 	path, err := os.Getwd()
// 	if err != nil {
// 		return fmt.Errorf("failed to open db: %w", err)
// 	}

// 	workingDir := fmt.Sprintf("%s/%s", path, db.workingDir)
// 	err = ensureDir(workingDir)
// 	if err != nil {
// 		return fmt.Errorf("failed to open db: %w", err)
// 	}

// 	db.i, err = sql.Open("dolt", fmt.Sprintf("file:///%s?commitname=Tester&commitemail=tester@test.com&database=%s", workingDir, dbName))
// 	if err != nil {
// 		return fmt.Errorf("failed to open db: %w", err)
// 	}

// 	return nil
// }

func (db *DB) Open() error {
	workingDir, err := filesys.LocalFS.Abs(db.workingDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", workingDir, err)
	}
	workingDirFS, err := filesys.LocalFS.WithWorkingDir(workingDir)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	err = ensureDir(workingDir)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	ctx := context.Background()
	dEnv := env.Load(ctx, env.GetCurrentUserHomeDir, workingDirFS, "file://"+workingDir+"/"+dbName, "1.6.1")
	err = dEnv.Config.WriteableConfig().SetStrings(map[string]string{
		env.UserEmailKey: "alex@giurgiu.io",
		env.UserNameKey:  "Alex Giurgiu",
	})
	if err != nil {
		return fmt.Errorf("failed to set config : %w", err)
	}

	db.mrEnv, err = env.MultiEnvForDirectory(ctx, dEnv.Config.WriteableConfig(), workingDirFS, dEnv.Version, dEnv.IgnoreLockFile, dEnv)
	if err != nil {
		return fmt.Errorf("failed to load database names")
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

	if dbEnv := db.mrEnv.GetEnv(dbName); dbEnv != nil {
		err = dbEnv.InitializeRepoState(ctx, "main")
		if err != nil {
			return fmt.Errorf("failed to init repo state: %w", err)
		}

		r := env.NewRemote("origin", fmt.Sprintf("protos://%s", dbName), map[string]string{})
		err = dbEnv.AddRemote(r)
		if err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	db.stopper = db.commitUpdater()

	return nil
}

func (db *DB) Close() error {
	err := db.sqle.Close()
	if err != context.Canceled {
		return err
	}

	if db.stopper != nil {
		db.stopper()
	}
	return nil
}

func (db *DB) Init() error {
	err := db.Query(fmt.Sprintf("CREATE DATABASE %s;", dbName), true)
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}

	err = db.mrEnv.Iter(func(name string, dEnv *env.DoltEnv) (stop bool, err error) {
		fmt.Println(name)
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate database names")
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

func (db *DB) Sync() error {
	return db.Query("CALL DOLT_PULL('origin', 'main');", true)
}

func (db *DB) GetDoltDB() (*doltdb.DoltDB, error) {
	testenv := db.mrEnv.GetEnv(dbName)
	if testenv == nil {
		return nil, fmt.Errorf("failed to retrieve db env")
	}

	return testenv.DoltDB, nil
}

func (db *DB) EnableP2P(p2p *p2p.P2P) error {
	db.p2p = p2p

	// dbdriver, ok := db.i.Driver().(*dd.DoltDriver)
	// if !ok {
	// 	return fmt.Errorf("SQL driver is not Dolt type")
	// }

	// doltDB, err := db.GetDoltDB()
	// if err != nil {
	// 	return fmt.Errorf("failed to open db: %w", err)
	// }
	// dbd := doltdb.HackDatasDatabaseFromDoltDB(doltDB)
	// cs := datas.ChunkStoreFromDatabase(dbd)
	// // dbdriver.RegisterDBFactory("protos", &dbclient.CustomFactory{P2P: db.p2p, LocalCS: cs})

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

func (db *DB) commitUpdater() func() error {
	updateTimer := time.NewTicker(1 * time.Second)
	commitTimmer := time.NewTicker(10 * time.Second)
	stopSignal := make(chan struct{})
	go func() {
		for {
			select {
			case <-updateTimer.C:
				commits, err := db.GetAllCommits()
				if err != nil {
					db.log.Errorf("failed to retrieve commits: %s", err.Error())
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

// func (db *DB) InitNew() error {
// 	workingDir, err := filesys.LocalFS.Abs(db.workingDir)
// 	if err != nil {
// 		return fmt.Errorf("failed to get absolute path for %s: %v", workingDir, err)
// 	}
// 	workingDirFS, err := filesys.LocalFS.WithWorkingDir(workingDir)
// 	if err != nil {
// 		return fmt.Errorf("failed to open db: %w", err)
// 	}

// 	// ensure working dir
// 	err = ensureDir(workingDir)
// 	if err != nil {
// 		return fmt.Errorf("failed to open db: %w", err)
// 	}

// 	ctx := context.Background()
// 	dEnv := env.Load(ctx, env.GetCurrentUserHomeDir, workingDirFS, "file://"+workingDir, "1.6.1")

// 	mrEnv, err := env.MultiEnvForDirectory(ctx, dEnv.Config.WriteableConfig(), workingDirFS, dEnv.Version, dEnv.IgnoreLockFile, dEnv)
// 	if err != nil {
// 		return fmt.Errorf("failed to load database names")
// 	}

// 	db.mrEnv = mrEnv

// 	config := &engine.SqlEngineConfig{
// 		IsReadOnly:              false,
// 		PrivFilePath:            ".doltcfg/privileges.db",
// 		BranchCtrlFilePath:      ".doltcfg/branch_control.db",
// 		DoltCfgDirPath:          ".doltcfg",
// 		ServerUser:              "root",
// 		ServerPass:              "",
// 		ServerHost:              "localhost",
// 		Autocommit:              true,
// 		DoltTransactionCommit:   false,
// 		JwksConfig:              []engine.JwksConfig{},
// 		ClusterController:       nil,
// 		BinlogReplicaController: binlogreplication.DoltBinlogReplicaController,
// 	}

// 	sqlEngine, err := engine.NewSqlEngine(context.Background(), db.mrEnv, config)
// 	if err != nil {
// 		return fmt.Errorf("failed to create sql engine: %w", err)
// 	}

// 	db.sqle = sqlEngine

// 	return nil
// }

// func (db *DB) PrintQueryResult(query string) {
// 	fmt.Println("query:", query)
// 	rows, err := db.i.Query(query)
// 	if err != nil {
// 		log.Fatalf("query '%s' failed: %s", query, err.Error())
// 	}
// 	defer rows.Close()

// 	fmt.Println("results:")
// 	err = printRows(rows, db.log)
// 	if err != nil {
// 		log.Fatalf("failed to print results for query '%s': %s", query, err.Error())
// 	}
// }
