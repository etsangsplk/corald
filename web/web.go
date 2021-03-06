package web

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/mycoralhealth/corald/auth0"
)

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, http.StatusNotFound, "")
}

// HandleError responds with the given HTTP response (and a generic message)
// and logs the long message to the log
func handleError(w http.ResponseWriter, r *http.Request, code int, long string) {
	http.Error(w, http.StatusText(code), code)
	log.Printf("%s %s: HTTP %d: %s", r.Method, r.URL, code, long)
}

func appendSlash(w http.ResponseWriter, r *http.Request) {
	u, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	u.Path = u.Path + "/"
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// MakeMuxRouter defines and creates routes
func MakeMuxRouter(dbCon *gorm.DB) http.Handler {

	// Wrapper to add dbCon to handler functions
	wrap := func(f func(w http.ResponseWriter, r *http.Request, dbCon *gorm.DB, u *auth0.UserInfo)) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			accessToken := r.Header.Get("X-Mycoral-Accesstoken")

			userInfo, err := auth0.Validate(accessToken)
			if err == auth0.Unauthorized {
				handleError(w, r, http.StatusUnauthorized, "")
				return
			} else if err != nil {
				handleError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			f(w, r, dbCon, &userInfo)
		}
	}

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/v0/session", wrap(handleSession)).Methods("GET")

	muxRouter.HandleFunc("/v0/ipfs/add", wrap(handleIPFSAdd)).Methods("POST")
	muxRouter.HandleFunc("/v0/ipfs/cat/{any:.*}", wrap(handleIPFSCat)).Methods("GET")

	/*
		* Currently unused
		muxRouter.HandleFunc("/v0/users", appendSlash).Methods("GET")
		muxRouter.HandleFunc("/v0/users/", wrap(handleGetAllUsers)).Methods("GET")
		muxRouter.HandleFunc("/v0/users/", wrap(handleCreateUser)).Methods("POST")
		muxRouter.HandleFunc("/v0/users/{userid}", wrap(handleGetUser)).Methods("GET")
		muxRouter.HandleFunc("/v0/users/{userid}", wrap(handleUpdateUser)).Methods("PUT")
		muxRouter.HandleFunc("/v0/users/{userid}", wrap(handleDeleteUser)).Methods("DELETE")
		muxRouter.HandleFunc("/{any:.*}", handleNotFound)
	*/

	return muxRouter
}

// Run starts server and app
func Run(dbCon *gorm.DB) error {

	httpAddr := os.Getenv("CORALD_ADDR")

	mux := MakeMuxRouter(dbCon)

	log.Printf("Listening on %s\n", httpAddr)
	s := &http.Server{
		Addr:           httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
