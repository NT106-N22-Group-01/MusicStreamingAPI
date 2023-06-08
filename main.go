// A web interface to your media library.
//
// This file is only here to make installing with go get easier.
// At the moment I don't see any other way to stash my source in the src directory
// instead of dumping it in the project root.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"NT106/Group01/MusicStreamingAPI/src"
)

var (
	// sqlFilesFS the migrations directory which contains SQL
	// migrations for sql-migrate and the initial schema. If the
	// embedded directory name changes, remember to change it in
	// main() too.
	//
	//go:embed sqls
	sqlFilesFS embed.FS
)

func main() {
	sqls, err := fs.Sub(sqlFilesFS, "sqls")
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading sqls subFS: %s\n", err)
		os.Exit(1)
	}

	src.Main(sqls)
}
