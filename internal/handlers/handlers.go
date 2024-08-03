package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/artem-streltsov/url-shortener/internal/database"
	"github.com/artem-streltsov/url-shortener/internal/safebrowsing"
	"github.com/artem-streltsov/url-shortener/internal/utils"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db        *database.DB
	templates *template.Template
	store     *sessions.CookieStore
}

func NewHandler(db *database.DB, secretKey string) *Handler {
	templatesDir := "./internal/templates"
	templates := template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html")))
	store := sessions.NewCookieStore([]byte(secretKey))
	return &Handler{db: db, templates: templates, store: store}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.indexHandler)
	mux.HandleFunc("/new", h.newURLHandler)
	mux.HandleFunc("/r/", h.redirectHandler)
	mux.HandleFunc("/register", h.registerHandler)
	mux.HandleFunc("/login", h.loginHandler)
	mux.HandleFunc("/logout", h.logoutHandler)
	mux.HandleFunc("/dashboard", h.dashboardHandler)
	mux.HandleFunc("/edit/", h.editURLHandler)
	mux.HandleFunc("/delete/", h.deleteURLHandler)
	return mux
}

func (h *Handler) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	session, _ := h.store.Get(r, "session")
	user, _ := session.Values["user"].(*database.User)

	flashes := session.Flashes("error")
	var errorMsg string
	if len(flashes) > 0 {
		errorMsg, _ = flashes[0].(string)
	}

	data := struct {
		User  *database.User
		Error string
	}{
		User:  user,
		Error: errorMsg,
	}
	session.Save(r, w)

	err := h.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) newURLHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "session")
	user, ok := session.Values["user"].(*database.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		err := h.templates.ExecuteTemplate(w, "new.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		url := r.Form.Get("url")
		password := r.Form.Get("password")

		if url == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		if !utils.IsValidURL(url) {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		isSafe, err := safebrowsing.IsSafeURL(url)
		if err != nil {
			log.Printf("Error checking URL safety: %v", err)
			// Continue with URL creation even if there's an error checking safety
		} else if !isSafe {
			http.Error(w, "The provided URL is not safe", http.StatusBadRequest)
			return
		}

		key := utils.GenerateKey(url)

		var hashedPassword string
		if password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Error hashing password", http.StatusInternalServerError)
				return
			}
			hashedPassword = string(hash)
		}

		if err := h.db.InsertURL(url, key, user.ID, hashedPassword); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) redirectHandler(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/r/")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	url, err := h.db.GetURL(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if url.Password != "" {
		switch r.Method {
		case http.MethodGet:
			err := h.templates.ExecuteTemplate(w, "password.html", struct{ Key string }{Key: key})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		case http.MethodPost:
			password := r.FormValue("password")
			if err := bcrypt.CompareHashAndPassword([]byte(url.Password), []byte(password)); err != nil {
				http.Error(w, "Invalid password", http.StatusUnauthorized)
				return
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	isSafe, err := safebrowsing.IsSafeURL(url.URL)
	if err != nil {
		log.Printf("Error checking URL safety: %v", err)
		// Continue with redirection even if there's an error checking safety
	} else if !isSafe {
		http.Error(w, "The requested URL is not safe", http.StatusForbidden)
		return
	}

	if err := h.db.IncrementClicks(url.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url.URL, http.StatusFound)
}

func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		err := h.templates.ExecuteTemplate(w, "register.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if username == "" || email == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		user, err := h.db.CreateUser(username, email, string(hashedPassword))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session, _ := h.store.Get(r, "session")
		session.Values["user"] = user
		err = session.Save(r, w)
		if err != nil {
			log.Printf("Error saving session: %v", err)
			http.Error(w, "Error saving session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "session")

	switch r.Method {
	case http.MethodGet:
		flashes := session.Flashes("error")
		var errorMsg string
		if len(flashes) > 0 {
			errorMsg, _ = flashes[0].(string)
		}
		session.Save(r, w)

		data := struct {
			Error string
		}{
			Error: errorMsg,
		}

		err := h.templates.ExecuteTemplate(w, "login.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		username := r.FormValue("username")
		password := r.FormValue("password")

		user, err := h.db.GetUserByUsername(username)
		if err != nil {
			session.AddFlash("Invalid username or password", "error")
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			session.AddFlash("Invalid username or password", "error")
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		session.Values["user"] = user
		err = session.Save(r, w)
		if err != nil {
			log.Printf("Error saving session: %v", err)
			http.Error(w, "Error saving session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "session")
	session.Values["user"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	session, err := h.store.Get(r, "session")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Error(w, "Error getting session", http.StatusInternalServerError)
		return
	}

	userValue, ok := session.Values["user"]
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, ok := userValue.(*database.User)
	if !ok {
		log.Printf("Error: user value is not of type *database.User")
		http.Error(w, "Invalid session data", http.StatusInternalServerError)
		return
	}

	urls, err := h.db.GetURLsByUserID(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		User *database.User
		URLs []database.URL
		Host string
	}{
		User: user,
		URLs: urls,
		Host: r.Host,
	}

	err = h.templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) editURLHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "session")
	user, ok := session.Values["user"].(*database.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	urlID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/edit/"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	url, err := h.db.GetURLByID(urlID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if url.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := struct {
			URL  *database.URL
			Host string
		}{
			URL:  url,
			Host: r.Host,
		}
		err := h.templates.ExecuteTemplate(w, "edit.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodPost:
		newURL := r.FormValue("url")
		newPassword := r.FormValue("password")

		if newURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		if !utils.IsValidURL(newURL) {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		isSafe, err := safebrowsing.IsSafeURL(newURL)
		if err != nil {
			log.Printf("Error checking URL safety: %v", err)
			// Continue with URL update even if there's an error checking safety
		} else if !isSafe {
			http.Error(w, "The provided URL is not safe", http.StatusBadRequest)
			return
		}

		var hashedPassword string
		if newPassword != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Error hashing password", http.StatusInternalServerError)
				return
			}
			hashedPassword = string(hash)
		}

		err = h.db.UpdateURL(urlID, newURL, hashedPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) deleteURLHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "session")
	user, ok := session.Values["user"].(*database.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	urlID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/delete/"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	url, err := h.db.GetURLByID(urlID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if url.UserID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	err = h.db.DeleteURL(urlID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
