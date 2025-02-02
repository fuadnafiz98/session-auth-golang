package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
)

type Template struct {
	tmpl *template.Template
}

func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// might have to use context
func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

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

func getSession(w http.ResponseWriter, r *http.Request) (s Session, e error) {
	cookie, err := r.Cookie("_session_id")

	if err != nil || cookie == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return Session{}, fmt.Errorf("No Session")
	}

	sessionId := cookie.Value

	store.mu.RLock()
	session, ok := store.sessions[sessionId]
	store.mu.RUnlock()

	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return Session{}, fmt.Errorf("No Session")
	}
	return session, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	template := newTemplate()
	session, err := getSession(w, r)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(session.Username)

	template.Render(w, "index.html", session)
}

func getLoginHandler(w http.ResponseWriter, r *http.Request) {
	template := newTemplate()
	template.Render(w, "login.html", nil)
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
	//declare template here
	router.HandleFunc("GET /", indexHandler)
	router.HandleFunc("GET /login", getLoginHandler)
	router.HandleFunc("POST /login", postLoginHandler)
	router.HandleFunc("GET /logout", logoutHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	fmt.Println("Server is running on port", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		panic("ERROR!")
	}
}
