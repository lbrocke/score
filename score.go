package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"score/src/parser"
	"strings"

	"github.com/biter777/countries"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thanhpk/randstr"
)

const (
	PATH_API    = "/api/"
	PATH_CLIENT = "/c/"
	PATH_INDEX  = "/"

	ACTION_NEW    = "new"
	ACTION_UPDATE = "update"

	// name of SQLite database
	DEFAULT_DB_PATH = "score.sqlite"

	// path to templates
	TEMPLATES = "tpl"

	// name of cookie that is used as client identification
	COOKIE_NAME = "token"
	// length of cookie value
	TOKEN_LENGTH = 64
)

var (
	//go:embed tpl/*
	files     embed.FS
	templates map[string]*template.Template
	database  string
)

type APIRequestData struct {
	Action string `json:"action"`
	Match  string `json:"match"`
	// match data is nested JSON, but must not be decoded automatically
	Data map[string]any `json:"data"`
}

type APIResponseData struct {
	Match string `json:"match"`
}

func initTemplates() error {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	entries, err := fs.ReadDir(files, TEMPLATES)
	if err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},
		"flag": func(country parser.Country) string {
			return countries.ByName(string(country)).Emoji()
		},
	}

	for _, tpl := range entries {
		if tpl.IsDir() {
			continue
		}

		pt, err := template.New(tpl.Name()).Funcs(funcMap).ParseFS(files, TEMPLATES+"/"+tpl.Name())
		if err != nil {
			return err
		}

		templates[tpl.Name()] = pt
	}

	return nil
}

func initDatabase() error {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return err
	}

	defer db.Close()

	stmt := `
		CREATE TABLE IF NOT EXISTS matches (
			uuid     TEXT NOT NULL PRIMARY KEY,
			token    TEXT,
			json     TEXT,
			modified DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TRIGGER IF NOT EXISTS update_modified AFTER UPDATE ON matches
		BEGIN
			UPDATE matches SET modified = datetime('now') WHERE json = NEW.json;
		END;
	`

	if _, err := db.Exec(stmt); err != nil {
		return err
	}

	return nil
}

func createMatch(token string) (string, error) {
	if len(token) == 0 {
		return "", errors.New("empty token")
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.New("cannot generate match uuid")
	}

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return "", errors.New("cannot open database")
	}

	defer db.Close()

	if _, err := db.Exec("INSERT INTO matches (uuid, token) VALUES (?, ?)", uuid.String(), token); err != nil {
		return "", errors.New("cannot create match")
	}

	return uuid.String(), nil
}

func updateMatch(raw string, m parser.Match, uuid string, token string) error {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return errors.New("cannot open database")
	}

	defer db.Close()

	newToken := sql.NullString{
		String: token,
		Valid:  true,
	}

	if m.Winner != parser.Unknown {
		// match is finished, delete token
		newToken.Valid = false
	}

	if _, err := db.Exec("UPDATE matches SET json = ?, token = ? WHERE uuid = ? AND token = ?",
		raw, newToken, uuid, token); err != nil {
		return errors.New("cannot update match")
	}

	return nil
}

func getRecentMatches() ([]parser.Match, error) {
	var matches []parser.Match

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return matches, err
	}

	defer db.Close()

	rows, err := db.Query("SELECT json FROM matches WHERE modified >= datetime('now', '-1 day') ORDER BY modified DESC")
	if err != nil {
		return matches, err
	}

	for rows.Next() {
		var json string

		rows.Scan(&json)

		match, err := parser.Parse(json)
		if err != nil {
			continue
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.EscapedPath()) != PATH_API {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := r.Cookie(COOKIE_NAME)
	if err != nil || len(token.Value) != TOKEN_LENGTH {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var requestData APIRequestData

	if err := json.Unmarshal(body, &requestData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch requestData.Action {
	case ACTION_NEW:
		uuid, err := createMatch(token.Value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponseData{
			Match: uuid,
		})
	case ACTION_UPDATE:
		// .Data is intentionally not un-marshalled into a struct yet,
		// but a map[string]any instead.
		// Combine into JSON and let the parser unmarshal it into a struct
		data, _ := json.Marshal(requestData.Data)

		match, err := parser.Parse(string(data))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if updateMatch(string(data), match, requestData.Match, token.Value) != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func handleClient(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.EscapedPath()) != PATH_CLIENT {
		http.Redirect(w, r, PATH_CLIENT, http.StatusSeeOther)
		return
	}

	token, err := r.Cookie(COOKIE_NAME)

	// set cookie if it does not exist yet
	if err != nil || len(token.Value) != TOKEN_LENGTH {
		http.SetCookie(w, &http.Cookie{
			Name:  COOKIE_NAME,
			Value: randstr.String(TOKEN_LENGTH),
			Path:  "/",
		})
	}

	t, ok := templates["client.html"]
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if err := t.Execute(w, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.EscapedPath()) != PATH_INDEX {
		http.Redirect(w, r, PATH_INDEX, http.StatusSeeOther)
		return
	}

	t, ok := templates["matches.html"]
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	matches, err := getRecentMatches()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := t.Execute(w, matches); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("Please provide the host/ip and port to listen on, e.g.\n\t$ score localhost:8080")
	}

	if path := os.Getenv("DB_PATH"); path != "" {
		database = path
	} else {
		database = DEFAULT_DB_PATH
	}

	if err := initTemplates(); err != nil {
		log.Fatalf("Could not load templates: %s\n", err)
	}

	if err := initDatabase(); err != nil {
		log.Fatalf("Could not load database: %s\n", err)
	}

	http.HandleFunc(PATH_API, handleAPI)
	http.HandleFunc(PATH_CLIENT, handleClient)
	http.HandleFunc(PATH_INDEX, handleIndex)

	log.Printf("Listening on http://%s\n", args[0])
	log.Fatal(http.ListenAndServe(args[0], nil))
}
