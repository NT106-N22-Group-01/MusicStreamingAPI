// Package webserver contains the webserver which deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"NT106/Group01/MusicStreamingAPI/src/config"
	"NT106/Group01/MusicStreamingAPI/src/library"
)

const (
	// notFoundAlbumImage is path to the image shown when there is no image
	// for particular album. It must be relative path in httpRootFS.
	notFoundAlbumImage = "images/unknownAlbum.png"
)

// User model
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique"`
	Password string
}

// Server represents our web server. It will be controlled from here
type Server struct {
	// Used for server-wide stopping, cancellation and stuff
	ctx context.Context

	// Calling this function will stop the server
	cancelFunc context.CancelFunc

	// Configuration of this server
	cfg config.Config

	// Makes sure Serve does not return before all the starting work ha been finished
	startWG sync.WaitGroup

	// The actual http.Server doing the HTTP work
	httpSrv *http.Server

	// The server's net.Listener. Used in the Server.Stop func
	listener net.Listener

	// This server's library with media
	library *library.LocalLibrary

	db *gorm.DB

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
	searchHandler := NewSearchHandler(srv.library)
	albumHandler := NewAlbumHandler(srv.library)
	artworkHandler := NewAlbumArtworkHandler(
		srv.library,
		notFoundAlbumImage,
	)
	artistImageHandler := NewArtistImagesHandler(srv.library)
	browseHandler := NewBrowseHandler(srv.library)
	mediaFileHandler := NewFileHandler(srv.library)
	loginTokenHandler := NewLoginTokenHandler(srv.db, srv.cfg.Secret)
	registerTokenHandler := NewRigisterTokenHandler(srv.db, srv.cfg.Secret)

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()

	// API methods
	router.Handle(APIv1EndpointSearchWithPath, searchHandler).Methods(
		APIv1Methods[APIv1EndpointSearchWithPath]...,
	)
	router.Handle(APIv1EndpointSearch, searchHandler).Methods(
		APIv1Methods[APIv1EndpointSearch]...,
	)
	router.Handle(APIv1EndpointDownloadAlbum, albumHandler).Methods(
		APIv1Methods[APIv1EndpointDownloadAlbum]...,
	)
	router.Handle(APIv1EndpointAlbumArtwork, artworkHandler).Methods(
		APIv1Methods[APIv1EndpointAlbumArtwork]...,
	)
	router.Handle(APIv1EndpointArtistImage, artistImageHandler).Methods(
		APIv1Methods[APIv1EndpointArtistImage]...,
	)
	router.Handle(APIv1EndpointBrowse, browseHandler).Methods(
		APIv1Methods[APIv1EndpointBrowse]...,
	)
	router.Handle(APIv1EndpointFile, mediaFileHandler).Methods(
		APIv1Methods[APIv1EndpointFile]...,
	)
	router.Handle(APIv1EndpointLoginToken, loginTokenHandler).Methods(
		APIv1Methods[APIv1EndpointLoginToken]...,
	)
	router.Handle(APIv1EndpointRegisterToken, registerTokenHandler).Methods(
		APIv1Methods[APIv1EndpointRegisterToken]...,
	)

	router.Handle("/search/{searchQuery}", searchHandler).Methods("GET")
	router.Handle("/search", searchHandler).Methods("GET")
	router.Handle("/album/{albumID}", albumHandler).Methods("GET")
	router.Handle("/album/{albumID}/artwork", artworkHandler).Methods(
		"GET", "PUT", "DELETE",
	)
	router.Handle("/artist/{artistID}/image", artistImageHandler).Methods(
		"GET", "PUT", "DELETE",
	)
	router.Handle("/file/{fileID}", mediaFileHandler).Methods("GET")
	router.Handle("/browse", browseHandler).Methods("GET")

	handler := NewTerryHandler(router)

	if srv.cfg.Auth {
		handler = NewAuthHandler(
			handler,
			srv.cfg.Secret,
			srv.db,
			[]string{
				"/v1/login/token/",
				"/login/",
				"/css/",
				"/js/",
				"/favicon/",
				"/fonts/",
			},
		)
	}

	handler = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, closeRequest := context.WithCancel(srv.ctx)
			h.ServeHTTP(w, r.WithContext(ctx))
			closeRequest()
		})
	}(handler)

	srv.httpSrv = &http.Server{
		Addr:           srv.cfg.Listen,
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   1200 * time.Second,
		MaxHeaderBytes: 1048576,
	}

	var reason error = srv.listenAndServe()

	log.Println("Webserver stopped.")

	if reason != nil {
		log.Printf("Reason: %s\n", reason)
	}

	srv.cancelFunc()
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
	cfg config.Config,
	lib *library.LocalLibrary,
	databasePath string,
) *Server {
	ctx, cancelCtx := context.WithCancel(ctx)

	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Perform automatic database migration
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	return &Server{
		ctx:        ctx,
		cancelFunc: cancelCtx,
		cfg:        cfg,
		library:    lib,
		db:         db,
	}
}
