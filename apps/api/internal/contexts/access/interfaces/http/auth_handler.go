package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/access/application"
	"github.com/jjspscl/my/internal/shared/response"
)

type AuthHandler struct {
	svc           *application.AuthService
	secureCookies bool
	sessionTTL    time.Duration
}

func NewAuthHandler(svc *application.AuthService, secureCookies bool, sessionTTL time.Duration) *AuthHandler {
	return &AuthHandler{
		svc:           svc,
		secureCookies: secureCookies,
		sessionTTL:    sessionTTL,
	}
}

func (h *AuthHandler) PublicRoutes(r chi.Router) {
	r.Post("/magic-link", h.RequestMagicLink)
	r.Post("/verify", h.VerifyToken)
	r.Get("/me", h.Me)
}

func (h *AuthHandler) ProtectedRoutes(r chi.Router) {
	r.Post("/logout", h.Logout)
}

type magicLinkRequest struct {
	Email string `json:"email"`
}

type verifyRequest struct {
	Token string `json:"token"`
}

type apiResponse struct {
	OK    bool   `json:"ok,omitempty"`
	Error string `json:"error,omitempty"`
}

func (h *AuthHandler) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var req magicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Email == "" {
		response.WriteError(w, r, http.StatusBadRequest, "email is required", nil)
		return
	}

	if err := h.svc.RequestMagicLink(r.Context(), req.Email); err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *AuthHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Token == "" {
		response.WriteError(w, r, http.StatusBadRequest, "token is required", nil)
		return
	}

	sessionID, err := h.svc.VerifyToken(r.Context(), req.Token)
	if err != nil {
		response.WriteError(w, r, http.StatusUnauthorized, err.Error(), err)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "my_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})

	// Set CSRF cookie (JS-readable)
	csrfToken := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "my_csrf",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("my_session")
	if err != nil {
		response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
		return
	}

	if err := h.svc.Logout(r.Context(), cookie.Value); err != nil {
		response.WriteError(w, r, http.StatusInternalServerError, "internal error", err)
		return
	}

	// Clear cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "my_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "my_csrf",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   h.secureCookies,
	})

	response.WriteJSON(w, http.StatusOK, apiResponse{OK: true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("my_session")
	if err != nil {
		response.WriteError(w, r, http.StatusUnauthorized, "not authenticated", err)
		return
	}

	email, err := h.svc.GetCurrentUser(r.Context(), cookie.Value)
	if err != nil {
		response.WriteError(w, r, http.StatusUnauthorized, "not authenticated", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"email": email})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
