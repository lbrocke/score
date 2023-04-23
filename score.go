package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"score/src/parser"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thanhpk/randstr"
)

const (
	ERR_INTERNAL_SERVER_ERROR = "Internal Server Error :("
)

const (
	DB_NAME = "score.sqlite"

	COOKIE_NAME  = "token"
	TOKEN_LENGTH = 64
)

type ResponseData struct {
	Message string `json:"message"`
}

func initDatabase() error {
	db, err := sql.Open("sqlite3", DB_NAME)

	if err != nil {
		return err
	}

	stmt := `
		CREATE TABLE IF NOT EXISTS matches (
			uuid     TEXT NOT NULL PRIMARY KEY,
			token    TEXT,
			running  BOOLEAN DEFAULT FALSE,
			json     TEXT,
			modified DATETIME DEFAULT (unixepoch())
		);

		CREATE TRIGGER IF NOT EXISTS update_modified AFTER UPDATE ON matches
		BEGIN
			UPDATE matches SET modified = unixepoch() WHERE json = NEW.json;
		END;
	`

	if _, err := db.Exec(stmt); err != nil {
		return err
	}

	return nil
}

func createNewMatch(token string) (string, error) {
	if len(token) == 0 {
		return "", errors.New("empty token")
	}

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		return "", errors.New("cannot open database")
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.New("cannot generate match uuid")
	}

	if _, err := db.Exec("INSERT INTO matches (uuid, token) VALUES (?, ?)", uuid.String(), token); err != nil {
		return "", errors.New("cannot create match")
	}

	return uuid.String(), nil
}

func updateMatch(raw string, m parser.Match, uuid string, token string) error {
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		return errors.New("cannot open database")
	}

	var (
		newToken = sql.NullString{
			String: token,
			Valid:  true,
		}
		running = true
	)

	if m.Winner != parser.Unknown {
		// match is finished, delete token
		newToken.Valid = false
		running = false
	}

	if _, err := db.Exec("UPDATE matches SET json = ?, token = ?, running = ? WHERE uuid = ? AND token = ?",
		raw, newToken, running, uuid, token); err != nil {
		return errors.New("cannot update match")
	}

	return nil
}

func respondWithJson(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")

	type ResponseData struct {
		Message string `json:"message"`
	}

	json.NewEncoder(w).Encode(ResponseData{
		Message: message,
	})
}

func handleMatchNew(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		http.Redirect(w, r, "/", http.StatusSeeOther)
	case http.MethodPost:
		token, err := r.Cookie(COOKIE_NAME)

		if err != nil || len(token.Value) != TOKEN_LENGTH {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if uuid, err := createNewMatch(token.Value); err == nil {
			w.WriteHeader(http.StatusCreated)
			respondWithJson(w, uuid)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleMatchExisting(w http.ResponseWriter, r *http.Request, uuid string) {
	switch r.Method {
	case http.MethodGet:
		// todo: check if uuid exists, return HTML
	case http.MethodPost:
		token, err := r.Cookie(COOKIE_NAME)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			respondWithJson(w, "No match data.")

			return
		}

		match, err := parser.Parse(string(body), true)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			respondWithJson(w, "Invalid match.")

			return
		}

		if updateMatch(string(body), match, uuid, token.Value) != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleMatch(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.EscapedPath(), "/m/")

	if len(uuid) == 0 {
		handleMatchNew(w, r)
	} else {
		handleMatchExisting(w, r, uuid)
	}
}

func handleClient(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie(COOKIE_NAME)

	// set cookie if it does not exist yet
	if err != nil || len(token.Value) != TOKEN_LENGTH {
		http.SetCookie(w, &http.Cookie{
			Name:  COOKIE_NAME,
			Value: randstr.String(TOKEN_LENGTH),
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// todo: send back client HTML/CSS/JS file(s)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// todo: send back HTML page with running matches, don't verify JSON
}

func main() {
	if err := initDatabase(); err != nil {
		log.Fatalf("Could not load database: %s", err)
	}

	http.HandleFunc("/m/", handleMatch)
	http.HandleFunc("/c/", handleClient)
	http.HandleFunc("/", handleIndex)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // todo: get port from argv
}
