package library

import (
	"database/sql"
	"fmt"
	"log"
)

func (lib *LocalLibrary) getTableSize(table string) int {
	var count int

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(fmt.Sprintf(`
            SELECT
                COUNT(*) as cnt
            FROM
                %s
        `, table))

		if err != nil {
			log.Printf("Query for getting %s count not prepared: %s\n", table, err)
			return nil
		}

		err = smt.QueryRow().Scan(&count)

		if err != nil {
			log.Printf("Query for getting %s count not successful: %s\n", table, err)
			return nil
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error getting table size query: %s", err)
		return count
	}

	return count
}
