package httpapi

import (
	"net"
	"net/http"
	"strings"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/database"
)

func (r *Router) handleRegister(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	user, err := r.auth.Register(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, user)
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.auth.Login(req.Context(), input.Username, input.Password, loginMetadataFromRequest(req))
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.Logout(req.Context(), token); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, user)
}

func (r *Router) authenticateBearerUser(req *http.Request) (database.User, string, error) {
	token, err := bearerToken(req)
	if err != nil {
		return database.User{}, "", err
	}
	user, err := r.auth.Authenticate(req.Context(), token)
	if err != nil {
		return database.User{}, "", err
	}
	return user, token, nil
}

func loginMetadataFromRequest(req *http.Request) auth.LoginMetadata {
	userAgent := strings.TrimSpace(req.UserAgent())
	return auth.LoginMetadata{
		UserAgent:  userAgent,
		RemoteAddr: requestRemoteAddr(req),
		DeviceName: deviceNameFromUserAgent(userAgent),
		ClientType: clientTypeFromUserAgent(userAgent),
	}
}

func requestRemoteAddr(req *http.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func deviceNameFromUserAgent(userAgent string) string {
	lower := strings.ToLower(userAgent)
	switch {
	case userAgent == "":
		return ""
	case strings.Contains(lower, "iphone"):
		return "iPhone"
	case strings.Contains(lower, "ipad"):
		return "iPad"
	case strings.Contains(lower, "android"):
		return "Android device"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		return "Mac"
	case strings.Contains(lower, "windows"):
		return "Windows PC"
	case strings.Contains(lower, "linux"):
		return "Linux device"
	default:
		return "Unknown device"
	}
}

func clientTypeFromUserAgent(userAgent string) string {
	if strings.TrimSpace(userAgent) == "" {
		return ""
	}
	return "Mibo Web"
}
