package webserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"gorm.io/gorm"
)

type loginTokenHandler struct {
	db     *gorm.DB
	secect string
}

var (
	rememberMeDuration = 62 * 24 * time.Hour
)

// NewLoginTokenHandler returns a new login handler which will use the information in
// auth for deciding when device or program was logged in correctly by entering
// username and password.
func NewLoginTokenHandler(db *gorm.DB, secect string) http.Handler {
	return &loginTokenHandler{
		db:     db,
		secect: secect,
	}
}

func (h *loginTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	reqBody := struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&reqBody); err != nil {
		respondWithJSONError(
			w,
			http.StatusBadRequest,
			"Error parsing JSON request: %s.",
			err,
		)
		return
	}

	if !checkLoginCreds(reqBody.User, reqBody.Pass, h.db) {
		respondWithJSONError(w, http.StatusUnauthorized, wrongLoginText)
		return
	}

	now := time.Now()
	pl := jwt.Payload{
		IssuedAt:       jwt.NumericDate(now),
		ExpirationTime: jwt.NumericDate(time.Now().Add(rememberMeDuration)),
	}

	if len(h.secect) == 0 {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error generating JWT: secret is empty.",
		)
		return
	}

	token, err := jwt.Sign(pl, jwt.NewHS256([]byte(h.secect)))
	if err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error generating JWT: %s.",
			err,
		)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(&struct {
		Token string `json:"token"`
	}{
		Token: string(token),
	})

	if err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error writing token response: %s.",
			err,
		)
		return
	}
}
