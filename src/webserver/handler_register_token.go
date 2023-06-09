package webserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type registerTokenHandler struct {
	db     *gorm.DB
	secect string
}

func NewRigisterTokenHandler(db *gorm.DB, secect string) http.Handler {
	return &registerTokenHandler{
		db:     db,
		secect: secect,
	}
}

func (register *registerTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	bytes, err := bcrypt.GenerateFromPassword([]byte(reqBody.Pass), 14)
	if err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error hash Password: %s.",
			err,
		)
		return
	}

	// Perform the registration logic here
	// You can use the reqBody.User and reqBody.Password to create a new user record in the database
	// and generate a registration token if needed

	// Example registration logic
	// create a new user in the database
	user := User{
		Username: reqBody.User,
		Password: string(bytes),
	}

	if err := register.db.Create(&user).Error; err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error saving user to database: %s.",
			err,
		)
		return
	}

	now := time.Now()
	pl := jwt.Payload{
		IssuedAt:       jwt.NumericDate(now),
		ExpirationTime: jwt.NumericDate(time.Now().Add(rememberMeDuration)),
	}

	if len(register.secect) == 0 {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error generating JWT: secret is empty.",
		)
		return
	}

	token, err := jwt.Sign(pl, jwt.NewHS256([]byte(register.secect)))
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
