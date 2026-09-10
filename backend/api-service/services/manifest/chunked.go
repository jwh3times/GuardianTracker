package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// chunkSize bounds how many hashes go into one IN-clause. SQLite caps the
// number of bound parameters per statement, and a cold collections fetch asks
// for thousands of definitions at once, so every batched read here is split.
const chunkSize = 500

// chunkedQuery describes one batched IN-query over a manifest definition table.
//
// Its three fields are exactly what the call sites differ by. Everything else —
// splitting the hashes, building the placeholder list, encoding the arguments,
// iterating rows, decoding the JSON blob, and closing each chunk — is identical
// across every batched read and lives in queryDefsChunked.
type chunkedQuery struct {
	// label prefixes every error this query returns. It names the operation the
	// caller invoked rather than the table, so an error still points at the call
	// site after passing through this shared helper.
	label string
	// statement builds the SQL for one chunk from its comma-joined placeholders.
	statement func(placeholders string) string
	// arg encodes one hash into the parameter the WHERE clause compares against.
	arg func(hash uint32) any
}

// byRowID matches definitions on the manifest's own id column, which stores a
// Bungie hash as a signed two's-complement int32 (see hashToDBKey).
func byRowID(table, label string) chunkedQuery {
	return chunkedQuery{
		label: label,
		statement: func(placeholders string) string {
			return "SELECT id, json FROM " + table + " WHERE id IN (" + placeholders + ")"
		},
		arg: func(hash uint32) any { return hashToDBKey(hash) },
	}
}

// queryDefsChunked runs q over hashes in IN-clause chunks, decoding each row's
// json column into T and handing it to collect together with the row's own id.
//
// A blob that fails to decode is skipped rather than failing the batch: the
// manifest is a third-party artifact, and one malformed definition must not
// deny the caller every other definition it asked for.
//
// collect receives the id so callers keyed on the row id (item lookups) and
// callers keyed on a field inside the blob (everything else) can share this
// path. It is called only for rows that decoded.
func queryDefsChunked[T any](db *sql.DB, hashes []uint32, q chunkedQuery, collect func(id uint32, def *T)) error {
	for i := 0; i < len(hashes); i += chunkSize {
		chunk := hashes[i:min(i+chunkSize, len(hashes))]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, hash := range chunk {
			placeholders[j] = "?"
			args[j] = q.arg(hash)
		}

		rows, err := db.Query(q.statement(strings.Join(placeholders, ",")), args...)
		if err != nil {
			return fmt.Errorf("%s: %w", q.label, err)
		}
		if err := collectChunk(rows, collect); err != nil {
			return fmt.Errorf("%s: %w", q.label, err)
		}
	}
	return nil
}

// collectChunk drains and closes one chunk's rows. It is a separate function so
// the close can be a defer: deferring inside queryDefsChunked's loop would hold
// every chunk's result set open until the whole batch finished.
func collectChunk[T any](rows *sql.Rows, collect func(id uint32, def *T)) error {
	defer rows.Close()
	for rows.Next() {
		var id int64
		var blob string
		if err := rows.Scan(&id, &blob); err != nil {
			return err
		}
		var def T
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		collect(dbKeyToHash(id), &def)
	}
	return rows.Err()
}
