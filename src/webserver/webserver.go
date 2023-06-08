// Package webserver contains the webserver which deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"NT106/Group01/MusicStreamingAPI/src/library"
)

const (
	// notFoundAlbumImage is path to the image shown when there is no image
	// for particular album. It must be relative path in httpRootFS.
	notFoundAlbumImage = "images/unknownAlbum.png"

	sessionCookieName  = "session"
	returnToQueryParam = "return_to"
)

// Server represents our web server. It will be controlled from here
type Server struct {
	// Used for server-wide stopping, cancellation and stuff
	ctx context.Context

	// Calling this function will stop the server
	cancelFunc context.CancelFunc

	// Configuration of this server
	// cfg config.Config

	// Makes sure Serve does not return before all the starting work ha been finished
	startWG sync.WaitGroup

	// The actual http.Server doing the HTTP work
	httpSrv *http.Server

	// The server's net.Listener. Used in the Server.Stop func
	listener net.Listener

	// This server's library with media
	library *library.LocalLibrary

	// htmlTemplatesFS is the directory with HTML templates.
	// htmlTemplatesFS fs.FS

	// httpRootFS is the directory which contains the
	// static files served by HTTPMS.
	httpRootFS fs.FS

	// Makes the server lockable. This lock should be used for accessing the
	// listener
	sync.Mutex
}

// Serve actually starts the webserver. It attaches all the handlers
// and starts the webserver while consulting the ServerConfig supplied. Trying to call
// this method more than once for the same server will result in panic.
func (srv *Server) Serve() {
	srv.Lock()
	defer srv.Unlock()
	if srv.listener != nil {
		panic("Second Server.Serve call for the same server")
	}
	srv.startWG.Add(1)
	go srv.serveGoroutine()
	srv.startWG.Wait()
}

func (srv *Server) serveGoroutine() {
	staticFilesHandler := http.FileServer(http.FS(srv.httpRootFS))
	searchHandler := NewSearchHandler(srv.library)

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()

	router.Handle(APIv1EndpointSearchWithPath, searchHandler).Methods(
		APIv1Methods[APIv1EndpointSearchWithPath]...,
	)

	router.Handle("/search/{searchQuery}", searchHandler).Methods("GET")
	router.Handle("/search", searchHandler).Methods("GET")
	router.PathPrefix("/").Handler(staticFilesHandler).Methods("GET")
}

// Uses our own listener to make our server stoppable. Similar to
// net.http.Server.ListenAndServer only this version saves a reference to the listener
func (srv *Server) listenAndServe() error {
	addr := srv.httpSrv.Addr
	if addr == "" {
		addr = "localhost:http"
	}
	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		srv.startWG.Done()
		return err
	}
	srv.listener = lsn
	log.Printf("Webserver started on http://%s\n", addr)
	srv.startWG.Done()
	return srv.httpSrv.Serve(lsn)
}

// Stop stops the webserver
func (srv *Server) Stop() {
	srv.Lock()
	defer srv.Unlock()
	if srv.listener != nil {
		srv.listener.Close()
		srv.listener = nil
	}
}

// Wait syncs whoever called this with the server's stop
func (srv *Server) Wait() {
	<-srv.ctx.Done()
}

// NewServer Returns a new Server using the supplied configuration cfg. The returned
// server is ready and calling its Serve method will start it.
func NewServer(
	ctx context.Context,
	//cfg config.Config,
	lib *library.LocalLibrary,
	httpRootFS fs.FS,
	htmlTemplatesFS fs.FS,
) *Server {
	ctx, cancelCtx := context.WithCancel(ctx)
	return &Server{
		ctx:        ctx,
		cancelFunc: cancelCtx,
		//cfg:             cfg,
		library:    lib,
		httpRootFS: httpRootFS,
		//htmlTemplatesFS: htmlTemplatesFS,
	}
}
