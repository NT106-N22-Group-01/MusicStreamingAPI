package webserver

import (
	"NT106/Group01/MusicStreamingAPI/src/library"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type FileHandlerCount struct {
	library library.Library
}

func (fh FileHandlerCount) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, fh.handleRequest)
}

// handleRequest handles the incoming HTTP request
func (fh FileHandlerCount) handleRequest(writer http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	case http.MethodGet:
		return fh.getFileCount(writer, req)
	default:
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}
}

// getFileCount retrieves the listen count for a media file
func (fh FileHandlerCount) getFileCount(writer http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	idStr := vars["fileID"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	if fh.library == nil {
		return fmt.Errorf("Library for FileHandler is nil")
	}

	count, err := fh.library.GetListenCount(id)
	if err != nil {
		http.Error(writer, "Failed to retrieve listens count", http.StatusInternalServerError)
		return nil
	}

	// Write the listen count as a response
	writer.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(writer, "%d", count)

	return nil
}

// NewFileHandler returns a new FileHandler that will be responsible for serving a file
// from the library identified by its ID.
func NewFileHandlerCount(lib library.Library) *FileHandlerCount {
	fh := new(FileHandlerCount)
	fh.library = lib
	return fh
}
