package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

const migrationTable = `
CREATE TABLE IF NOT EXISTS migrations (
	id SERIAL PRIMARY KEY,
	name TEXT UNIQUE NOT NULL,
	applied_at TIMESTAMP DEFAULT NOW()
);
`

func Run(db *sql.DB, dir string) error {

	_, err := db.Exec(migrationTable)
	if err != nil {
		return fmt.Errorf("failed creating migrations table: %w", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed reading migrations dir: %w", err)
	}

	var migrationFiles []string

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			migrationFiles = append(migrationFiles, f.Name())
		}
	}

	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {

		var exists bool

		err := db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM migrations WHERE name=$1)",
			file,
		).Scan(&exists)

		if err != nil {
			return err
		}

		if exists {
			continue
		}

		log.Printf("running migration: %s", file)

		content, err := ioutil.ReadFile(dir + "/" + file)
		if err != nil {
			return err
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed executing %s: %w", file, err)
		}

		_, err = db.Exec(
			"INSERT INTO migrations (name) VALUES ($1)",
			file,
		)
		if err != nil {
			return err
		}
	}

	return nil
}