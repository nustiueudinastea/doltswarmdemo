package main

import (
	"fmt"
	"time"

	"github.com/segmentio/ksuid"
)

const (
	tableName = "testtable"
)

func insert(data string) error {
	uid, err := ksuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to create uid: %w", err)
	}

	queryString := fmt.Sprintf("INSERT INTO %s (id, name) VALUES ('%s', '%s');", tableName, uid.String(), data)

	tx, err := dbi.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.Exec(queryString)
	if err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	// add
	_, err = tx.Exec(`CALL DOLT_ADD('-A');`)
	if err != nil {
		return fmt.Errorf("failed to commit table: %w", err)
	}

	// commit
	_, err = tx.Exec(fmt.Sprintf("CALL DOLT_COMMIT('-m', 'Inserted time', '--author', 'Alex Giurgiu <alex@giurgiu.io>', '--date', '%s');", time.Now().Format(time.RFC3339Nano)))
	if err != nil {
		return fmt.Errorf("failed to commit table: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit insert transaction: %w", err)
	}

	// async advertise new head
	go dbi.AdvertiseHead()

	return nil
}
