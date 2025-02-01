package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type Session struct {
	Username string
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

// a global session store for now
// TODO: have to make a persistant store

var store = SessionStore{
	sessions: make(map[string]Session),
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// try to get the session
	cookie, err := r.Cookie("_session_id")

	if err != nil || cookie == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sessionId := cookie.Value

	store.mu.RLock()
	session, ok := store.sessions[sessionId]
	store.mu.RUnlock()

	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	response := `
	<html>
		<head>
			<title>Go Session Auth</title>
		</head>
		<body>
			<h1>Welcome!</h1>
	`
	response += session.Username + "<br />"
	response += `
	<a href="/logout">Logout</a>
		</body>
	</html>
	`
	fmt.Fprintf(w, "%v", response)
}

func getLoginHandler(w http.ResponseWriter, r *http.Request) {
	response := `
	<html>
		<head>
			<title>Go Session Auth</title>
		</head>
		<body>
			<form action="/login" method="POST">
				<label for="username">Username:</label>
				<input type="text" name="username" id="username" /><br />
				<label for="password">Password:</label>
				<input type="password" name="password" id="password" /><br />
				<button class="btn" type="submit">Submit</button>
		</body>
	</html>
	`
	fmt.Fprintf(w, "%v", response)
}

func generateSessionId() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func postLoginHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != "admin" || password != "admin" {
		http.Error(w, "Wrong Credentials", http.StatusBadRequest)
		return
	}

	sessionId, err := generateSessionId()

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	store.mu.Lock()
	store.sessions[sessionId] = Session{
		Username: username,
	}
	store.mu.Unlock()

	cookie := http.Cookie{
		Name:  "_session_id",
		Value: sessionId,
		Path:  "/",
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("_session_id")

	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	sessionId := cookie.Value

	store.mu.Lock()
	delete(store.sessions, sessionId)
	store.mu.Unlock()

	cookie.MaxAge = -1
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {

	router := http.NewServeMux()
	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("GET /login", getLoginHandler)
	router.HandleFunc("POST /login", postLoginHandler)
	router.HandleFunc("GET /logout", logoutHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("--- SERVER IS RUNNING ---")
	err := server.ListenAndServe()
	if err != nil {
		panic("ERROR!")
	}

}
