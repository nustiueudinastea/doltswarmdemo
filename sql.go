package main

import (
	"fmt"

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
	_, err = dbi.Exec(queryString)
	if err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	// async advertise new head
	go dbi.AdvertiseHead()

	return nil
}
