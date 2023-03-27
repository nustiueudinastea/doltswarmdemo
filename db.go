package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	dd "github.com/dolthub/driver"
	"github.com/segmentio/ksuid"
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

var dbName = "test"

type DB struct {
	i              *sql.DB
	stopper        func() error
	commitListChan chan []Commit
}

func ensureDir(dirName string) error {
	err := os.Mkdir(dirName, os.ModePerm)
	if err == nil {
		return nil
	}
	if os.IsExist(err) {
		info, err := os.Stat(dirName)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("path exists but is not a directory")
		}
		return nil
	}
	return err
}

func (db *DB) Open(dir string, commitListChan chan []Commit) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	workingDir := fmt.Sprintf("%s/%s", path, dir)

	err = ensureDir(workingDir)
	if err != nil {
		return err
	}

	dbi, err := sql.Open("dolt", fmt.Sprintf("file:///%s?commitname=Tester&commitemail=tester@test.com&database=%s", workingDir, dbName))
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	dbdriver, ok := dbi.Driver().(*dd.DoltDriver)
	if !ok {
		return fmt.Errorf("SQL driver is not Dolt type")
	}
	dbdriver.RegisterDBFactory("protos", &ProtosFactory{})

	db.i = dbi

	_, err = db.i.Query("CREATE DATABASE IF NOT EXISTS test;")
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}

	_, err = db.i.Query("USE test;")
	if err != nil {
		return fmt.Errorf("failed to use db: %w", err)
	}

	_, err = db.i.Query("CALL DOLT_REMOTE('add','origin','protos://test');")
	if err != nil {
		return fmt.Errorf("failed to add remote db: %w", err)
	}

	// db.PrintQueryResult("CALL DOLT_PULL('origin', 'main');")

	// if err != nil {
	// 	if !strings.Contains(err.Error(), "database not found") {
	// 		return fmt.Errorf("failed to use db: %w", err)
	// 	}

	// 	mre, err := db.GetDoltMultiRepoEnv()
	// 	if err != nil {
	// 		return fmt.Errorf("failed to retrieve db env: %w", err)
	// 	}

	// 	dbDir := workingDir + "/test"
	// 	err = ensureDir(dbDir)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	localFS, err := filesys.LocalFilesysWithWorkingDir(dbDir)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create local fs for db env: %w", err)
	// 	}

	// 	fmt.Println(doltdb.LocalDirDoltDB)
	// 	newEnv := env.Load(context.TODO(), env.GetCurrentUserHomeDir, localFS, doltdb.LocalDirDoltDB, "test")
	// 	err = newEnv.InitRepoWithTime(context.TODO(), types.Format_DOLT_1, "test", "alex@giurgiu.io", "main", time.Date(2022, 11, 1, 0, 0, 0, 0, time.UTC))
	// 	if err != nil {
	// 		return fmt.Errorf("failed to init db env: %w", err)
	// 	}

	// 	mre.AddEnv("test", newEnv)

	// }

	err = db.query("SET @@dolt_transaction_commit = 1;")
	if err != nil {
		return fmt.Errorf("failed to set transaction commit: %w", err)
	}

	query := `CREATE TABLE IF NOT EXISTS protos(
		id varchar(256) PRIMARY KEY,
		name varchar(512)
	);`

	err = db.query(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	commits, err := db.GetAllCommits()
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if len(commits) == 0 {
		db.Insert("main", "first")
	}

	db.commitListChan = commitListChan
	db.stopper = db.commitUpdater()

	return nil
}

func (db *DB) Close() error {
	db.stopper()
	return db.i.Close()
}

func (db *DB) query(query string) error {
	rows, err := db.i.Query(query)
	if err != nil {
		return fmt.Errorf("query '%s' failed: %w", query, err)
	}
	defer rows.Close()
	return nil
}

func (db *DB) Insert(branch string, data string) error {
	uid, err := ksuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to create uid: %w", err)
	}

	queryString := fmt.Sprintf("INSERT INTO `%s/%s`.protos (id, name) VALUES {};", dbName, branch)
	_, err = sq.Exec(db.i, sq.
		Queryf(queryString, sq.RowValue{
			uid.String(), data,
		}).
		SetDialect(sq.DialectMySQL),
	)
	if err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	db.query(fmt.Sprintf("SELECT * FROM `%s/%s`.protos;", dbName, branch))

	return nil
}

func (db *DB) PrintQueryResult(query string) {
	fmt.Println("query:", query)
	rows, err := db.i.Query(query)
	fmt.Println(rows)
	fmt.Println(err)
	if err != nil {
		log.Fatalf("query '%s' failed: %s", query, err.Error())
	}
	defer rows.Close()

	fmt.Println("results:")
	err = printRows(rows)
	if err != nil {
		log.Fatalf("failed to print results for query '%s': %s", query, err.Error())
	}

	fmt.Println()
}

func commitMapper(row *sq.Row) (Commit, error) {
	commit := Commit{
		Hash:         row.String("commit_hash"),
		Table:        row.String("table_name"),
		Committer:    row.String("committer"),
		Email:        row.String("email"),
		Date:         row.Time("date"),
		Message:      row.String("message"),
		DataChange:   row.Bool("data_change"),
		SchemaChange: row.Bool("schema_change"),
	}
	return commit, nil
}

func (db *DB) GetAllCommits() ([]Commit, error) {
	commits, err := sq.FetchAll(db.i, sq.
		Queryf("SELECT {*} FROM dolt_diff ORDER BY date;").
		SetDialect(sq.DialectMySQL),
		commitMapper,
	)
	if err != nil {
		return commits, fmt.Errorf("failed to retrieve last commit hash: %w", err)
	}

	if len(commits) == 0 {
		return commits, fmt.Errorf("no commits found")
	}

	return commits, nil
}

func (db *DB) GetLastCommit() (Commit, error) {
	commits, err := sq.FetchAll(db.i, sq.
		Queryf("SELECT {*} FROM dolt_diff ORDER BY date DESC LIMIT 1;").
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

func (db *DB) GetDoltDB() (*doltdb.DoltDB, error) {
	dc := db.i.Driver()
	dbdriver, ok := dc.(*dd.DoltDriver)
	if !ok {
		return nil, fmt.Errorf("SQL driver is not Dolt type")
	}

	mrEnv := dbdriver.GetMREnv()
	testenv := mrEnv.GetEnv("test")
	if testenv == nil {
		return nil, fmt.Errorf("failed to retrieve db env")
	}

	return testenv.DoltDB, nil
}

func (db *DB) GetDoltMultiRepoEnv() (*env.MultiRepoEnv, error) {
	dc := db.i.Driver()
	dbdriver, ok := dc.(*dd.DoltDriver)
	if !ok {
		return nil, fmt.Errorf("SQL driver is not Dolt type")
	}

	return dbdriver.GetMREnv(), nil
}

func printRows(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	fmt.Println(strings.Join(cols, "|"))

	for rows.Next() {
		values := make([]interface{}, len(cols))
		var generic = reflect.TypeOf(values).Elem()
		for i := 0; i < len(cols); i++ {
			values[i] = reflect.New(generic).Interface()
		}

		err = rows.Scan(values...)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		result := bytes.NewBuffer(nil)
		for i := 0; i < len(cols); i++ {
			if i != 0 {
				result.WriteString("|")
			}

			var rawValue = *(values[i].(*interface{}))
			switch val := rawValue.(type) {
			case string:
				result.WriteString(val)
			case int:
				result.WriteString(strconv.FormatInt(int64(val), 10))
			case int8:
				result.WriteString(strconv.FormatInt(int64(val), 10))
			case int16:
				result.WriteString(strconv.FormatInt(int64(val), 10))
			case int32:
				result.WriteString(strconv.FormatInt(int64(val), 10))
			case int64:
				result.WriteString(strconv.FormatInt(val, 10))
			case uint:
				result.WriteString(strconv.FormatUint(uint64(val), 10))
			case uint8:
				result.WriteString(strconv.FormatUint(uint64(val), 10))
			case uint16:
				result.WriteString(strconv.FormatUint(uint64(val), 10))
			case uint32:
				result.WriteString(strconv.FormatUint(uint64(val), 10))
			case uint64:
				result.WriteString(strconv.FormatUint(val, 10))
			case float32:
				result.WriteString(strconv.FormatFloat(float64(val), 'f', 2, 64))
			case float64:
				result.WriteString(strconv.FormatFloat(val, 'f', 2, 64))
			case bool:
				if val {
					result.WriteString("true")
				} else {
					result.WriteString("false")
				}
			case []byte:
				enc := base64.NewEncoder(base64.URLEncoding, result)
				_, err := enc.Write(val)
				return fmt.Errorf("failed to base64 encode blob: %w", err)
			case time.Time:
				timeStr := val.Format(time.RFC3339)
				result.WriteString(timeStr)
			}
		}

		fmt.Println(result.String())
	}

	return nil
}

func (db *DB) commitUpdater() func() error {
	timer := time.NewTimer(1 * time.Second)
	stopSignal := make(chan struct{})
	go func() {
		log.Info("Starting commit updater")
		for {
			select {
			case <-timer.C:
				commits, err := db.GetAllCommits()
				if err != nil {
					log.Error("failed to retrieve commits")
					continue
				}
				db.commitListChan <- commits
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
