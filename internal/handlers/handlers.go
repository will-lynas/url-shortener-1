package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/artem-streltsov/url-shortener/internal/auth"
	"github.com/artem-streltsov/url-shortener/internal/database"
	"github.com/artem-streltsov/url-shortener/internal/safebrowsing"
	"github.com/artem-streltsov/url-shortener/internal/utils"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Routes() http.Handler {
	commonMiddleware := alice.New(h.loggingMiddleware)
	authMiddleware := commonMiddleware.Append(h.authMiddleware)
	router := mux.NewRouter()

	router.Handle("/register", commonMiddleware.ThenFunc(h.registerHandler)).Methods("POST")
	router.Handle("/login", commonMiddleware.ThenFunc(h.loginHandler)).Methods("POST")
	router.Handle("/r/{key}", commonMiddleware.ThenFunc(h.redirectHandler)).Methods("POST")

	router.Handle("/dashboard", authMiddleware.ThenFunc(h.dashboardHandler)).Methods("GET")
	router.Handle("/new", authMiddleware.ThenFunc(h.newURLHandler)).Methods("POST")
	router.Handle("/edit/{id}", authMiddleware.ThenFunc(h.editURLHandler)).Methods("PUT")
	router.Handle("/delete/{id}", authMiddleware.ThenFunc(h.deleteURLHandler)).Methods("DELETE")

	return router
}

func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrTokenExpired {
				newToken, refreshErr := auth.RefreshToken(tokenString)
				if refreshErr != nil {
					http.Error(w, "Could not refresh token", http.StatusUnauthorized)
					return
				}
				w.Header().Set("X-Auth-Token", newToken)
				claims, _ = auth.ValidateToken(newToken)
			} else {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
		}

		user, err := h.db.GetUserByID(claims.UserID)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		ctx := utils.ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	email := r.Form.Get("email")
	password := r.Form.Get("password")

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
		http.Error(w, fmt.Sprintf("Error creating user: %v", err), http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating token: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Auth-Token", token)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	password := r.Form.Get("password")

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Auth-Token", token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := utils.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.db.GetURLsByUserID(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    if urls == nil {
        urls = []database.URL{}
    }

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string][]database.URL{"urls": urls})
}

func (h *Handler) newURLHandler(w http.ResponseWriter, r *http.Request) {
	user := utils.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	correctedURL, isValid := utils.IsValidURL(url)
	if !isValid {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	isSafe, err := safebrowsing.IsSafeURL(correctedURL)
	if err != nil {
		http.Error(w, "Error checking URL safety", http.StatusBadRequest)
		return
	}
	if !isSafe {
		http.Error(w, "The provided URL is not safe", http.StatusBadRequest)
		return
	}

	key := utils.GenerateKey(correctedURL)

	if err := h.db.InsertURL(correctedURL, key, user.ID, password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func (h *Handler) redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	url, err := h.db.GetURLByKey(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if url.Password != "" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}

		password := r.Form.Get("password")

		if password == "" {
			http.Error(w, "Password is required", http.StatusBadRequest)
			return
		}

		if password != url.Password {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	isSafe, err := safebrowsing.IsSafeURL(url.URL)
	if err != nil {
		http.Error(w, "Error checking URL safety", http.StatusForbidden)
		return
	}
	if !isSafe {
		http.Error(w, "The requested URL is not safe", http.StatusForbidden)
		return
	}

	if err := h.db.IncrementClicks(url.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"url": url.URL})
}

func (h *Handler) editURLHandler(w http.ResponseWriter, r *http.Request) {
	user := utils.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	urlIDStr := vars["id"]
	urlID, err := strconv.ParseInt(urlIDStr, 10, 64)
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

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	newURL := r.Form.Get("url")
	newPassword := r.Form.Get("password")

	if newURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	correctedURL, isValid := utils.IsValidURL(newURL)
	if !isValid {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	isSafe, err := safebrowsing.IsSafeURL(correctedURL)
	if err != nil {
		http.Error(w, "Error checking URL safety", http.StatusBadRequest)
		return
	}
	if !isSafe {
		http.Error(w, "The provided URL is not safe", http.StatusBadRequest)
		return
	}

	err = h.db.UpdateURL(urlID, correctedURL, newPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteURLHandler(w http.ResponseWriter, r *http.Request) {
	user := utils.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	urlIDStr := vars["id"]
	urlID, err := strconv.ParseInt(urlIDStr, 10, 64)
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

	w.WriteHeader(http.StatusOK)
}
