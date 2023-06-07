package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/howeyc/fsnotify"

	// Blind import is the way a SQL driver is imported. This is the proposed way
	// from the golang documentation.
	_ "github.com/mattn/go-sqlite3"

	"NT106/Group01/MusicStreamingAPI/src/art"
	"NT106/Group01/MusicStreamingAPI/src/scaler"
)

const (
	// UnknownLabel will be used in case some media tag is missing. As a consequence
	// if there are many files with missing title, artist and album only
	// one of them will be saved in the library.
	UnknownLabel = "Unknown"

	// SQLiteMemoryFile can be used as a database path for the sqlite's Open method.
	// When using it, one would create a memory database which does not write
	// anything on disk. See https://www.sqlite.org/inmemorydb.html for more info
	// on the subject of in-memory databases. We are using a shared cache because
	// this causes all the different connections in the database/sql pool to be
	// connected to the same "memory file". Without this. every new connection
	// would end up creating a new memory database.
	SQLiteMemoryFile = "file::memory:?cache=shared"

	// sqlSchemaFile is the file which contains the initial SQL Schema for the
	// media library. It must be one of the files in `sqlFilesFS`.
	sqlSchemaFile = "library_schema.sql"
)

var (
	// ErrAlbumNotFound is returned when no album could be found for particular operation.
	ErrAlbumNotFound = errors.New("Album Not Found")

	// ErrArtistNotFound is returned when no artist could be found for particular operation.
	ErrArtistNotFound = errors.New("Artist Not Found")

	// ErrArtworkNotFound is returned when no artwork can be found for particular album.
	ErrArtworkNotFound = NewArtworkError("Artwork Not Found")

	// ErrCachedArtworkNotFound is returned when the database has been queried and
	// its cache says the artwork was not found in the recent past. No need to continue
	// searching further once you receive this error.
	ErrCachedArtworkNotFound = NewArtworkError("Artwork Not Found (Cached)")

	// ErrArtworkTooBig is returned from operation when the artwork is too big for it to
	// handle.
	ErrArtworkTooBig = NewArtworkError("Artwork Is Too Big")
)

// ArtworkError represents some kind of artwork error.
type ArtworkError struct {
	Err string
}

// Error implements the error interface.
func (a *ArtworkError) Error() string {
	return a.Err
}

// NewArtworkError returns a new artwork error which will have `err` as message.
func NewArtworkError(err string) *ArtworkError {
	return &ArtworkError{Err: err}
}

func init() {
}

// LocalLibrary implements the Library interface. Will represent files found on the
// local storage
type LocalLibrary struct {
	// The configuration for how to scan the libraries.
	// ScanConfig config.ScanSection

	database string         // The location of the library's database
	paths    []string       // FS locations which contain the library's media files
	db       *sql.DB        // Database handler
	walkWG   sync.WaitGroup // Used to log how much time scanning took

	// If something needs to work with the database it has to construct
	// a DatabaseExecutable and send it through this channel.
	dbExecutes chan DatabaseExecutable

	// artworkSem is used to make sure there are no more than certain amount
	// of artwork resolution tasks at a given moment.
	artworkSem chan struct{}

	// Directory watcher
	watch     *fsnotify.Watcher
	watchLock *sync.RWMutex

	ctx           context.Context
	ctxCancelFunc context.CancelFunc

	waitScanLock sync.RWMutex

	artFinder art.Finder

	fs         fs.FS
	sqlFilesFS fs.FS

	imageScaler scaler.Scaler

	// cleanupLock is used to secure a thread safe access to the runningCleanup property.
	cleanupLock *sync.RWMutex

	// runningCleanup shows whether there is an already running clean-up.
	runningCleanup bool

	// runningRescan shows that at the moment a complete rescan is running.
	runningRescan bool
}

// Close closes the database connection. It is safe to call it as many times as you want.
func (lib *LocalLibrary) Close() {
	lib.ctxCancelFunc()
	lib.db.Close()
}

// AddLibraryPath adds a library directory to the list of libraries which will be
// scanned and consequently watched.
func (lib *LocalLibrary) AddLibraryPath(path string) {
	if _, err := fs.Stat(lib.fs, path); err != nil {
		log.Printf("error adding path: %s", err)
		return
	}

	lib.paths = append(lib.paths, path)
}

// Search searches in the library. Will match against the track's name, artist and album.
func (lib *LocalLibrary) Search(searchTerm string) []SearchResult {
	searchTerm = fmt.Sprintf("%%%s%%", searchTerm)

	var output []SearchResult
	work := func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT
				t.id as track_id,
				t.name as track,
				al.name as album,
				at.name as artist,
				at.id as artist_id,
				t.number as track_number,
				t.album_id as album_id,
				t.fs_path as fs_path,
				t.duration as duration
			FROM
				tracks as t
					LEFT JOIN albums as al ON al.id = t.album_id
					LEFT JOIN artists as at ON at.id = t.artist_id
			WHERE
				t.name LIKE ? OR
				al.name LIKE ? OR
				at.name LIKE ?
			ORDER BY
				al.name, t.number
		`, searchTerm, searchTerm, searchTerm)
		if err != nil {
			log.Printf("Query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res SearchResult

			err := rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
				&res.ArtistID, &res.TrackNumber, &res.AlbumID, &res.Format,
				&res.Duration)
			if err != nil {
				log.Printf("Error scanning search result: %s\n", err)
				continue
			}

			res.Format = mediaFormatFromFileName(res.Format)

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing search db work: %s", err)
		return output
	}
	return output
}

// GetFilePath returns the filesystem path for a file specified by its ID.
func (lib *LocalLibrary) GetFilePath(ID int64) string {
	var filePath string
	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				fs_path
			FROM
				tracks
			WHERE
				id = ?
		`)
		if err != nil {
			log.Printf("Error getting file path: %s\n", err)
			return nil
		}

		defer smt.Close()

		err = smt.QueryRow(ID).Scan(&filePath)
		if err != nil {
			log.Printf("Error file path query row: %s\n", err)
			return nil
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing get file path db work: %s", err)
		return filePath
	}
	return filePath
}

// GetAlbumFiles satisfies the Library interface
func (lib *LocalLibrary) GetAlbumFiles(albumID int64) []SearchResult {
	var output []SearchResult
	work := func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT
				t.id as track_id,
				t.name as track,
				al.name as album,
				at.name as artist,
				at.id as artist_id,
				t.number as track_number,
				t.album_id as album_id,
				t.fs_path as fs_path
			FROM
				tracks as t
					LEFT JOIN albums as al ON al.id = t.album_id
					LEFT JOIN artists as at ON at.id = t.artist_id
			WHERE
				t.album_id = ?
			ORDER BY
				al.name, t.number
		`, albumID)
		if err != nil {
			log.Printf("Query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res SearchResult
			err := rows.Scan(
				&res.ID,
				&res.Title,
				&res.Album,
				&res.Artist,
				&res.ArtistID,
				&res.TrackNumber,
				&res.AlbumID,
				&res.Format,
			)
			if err != nil {
				return fmt.Errorf("scanning error: %w", err)
			}

			res.Format = mediaFormatFromFileName(res.Format)

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing get album files db work: %s", err)
		return output
	}
	return output
}

// Removes the file from the library. That means finding it in the database and
// removing it from there.
func (lib *LocalLibrary) removeFile(filePath string) {

	fullPath, err := filepath.Abs(filePath)

	if err != nil {
		log.Printf("Error removing %s: %s\n", filePath, err.Error())
		return
	}

	work := func(db *sql.DB) error {
		_, err := db.Exec(`
			DELETE FROM tracks
			WHERE fs_path = ?
		`, fullPath)
		if err != nil {
			log.Printf("Error removing %s: %s\n", fullPath, err.Error())
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing remove file db work: %s", err)
	}
}

// Removes files which belong in this directory from the library.
func (lib *LocalLibrary) removeDirectory(dirPath string) {

	// Adding slash at the end to make sure we are always removing directories
	deleteMatch := fmt.Sprintf("%s/%%", strings.TrimRight(dirPath, "/"))

	work := func(db *sql.DB) error {
		_, err := db.Exec(`
			DELETE FROM tracks
			WHERE fs_path LIKE ?
		`, deleteMatch)
		if err != nil {
			log.Printf("Error removing %s: %s\n", dirPath, err.Error())
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing remove dir db work: %s", err)
	}
}

// NewLocalLibrary returns a new LocalLibrary which will use for database the file
// specified by databasePath. Also creates the database connection so you does not
// need to worry about that. It accepts the parent's context and create its own
// child context.
func NewLocalLibrary(
	ctx context.Context,
	databasePath string,
	sqlFilesFS fs.FS,
) (*LocalLibrary, error) {
	lib := new(LocalLibrary)
	lib.database = databasePath
	lib.sqlFilesFS = sqlFilesFS
	lib.fs = &osFS{}

	libContext, cancelFunc := context.WithCancel(ctx)

	lib.ctx = libContext
	lib.ctxCancelFunc = cancelFunc

	var err error

	lib.db, err = sql.Open("sqlite3", lib.database)

	if err != nil {
		return nil, err
	}

	lib.watchLock = &sync.RWMutex{}
	lib.artworkSem = make(chan struct{}, 10)

	lib.cleanupLock = &sync.RWMutex{}

	var wg sync.WaitGroup
	wg.Add(1)
	go lib.databaseWorker(&wg)
	wg.Wait()

	return lib, nil
}

const thumbnailWidth = 60
