package db

import "github.com/bokwoon95/sq"

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
