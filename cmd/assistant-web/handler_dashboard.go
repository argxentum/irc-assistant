package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dashboardSessionCookie = "dashboard_session"

type dashboardSession struct {
	Nick    string
	Channel string
}

func (s *server) signSessionCookie(nick, channel string, expiry time.Time) string {
	payload := fmt.Sprintf("%s|%s|%d", nick, channel, expiry.Unix())
	mac := hmac.New(sha256.New, []byte(s.cfg.Web.SessionSecret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

func (s *server) verifySessionCookie(value string) *dashboardSession {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil
	}

	mac := hmac.New(sha256.New, []byte(s.cfg.Web.SessionSecret))
	mac.Write(payload)
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
		return nil
	}

	fields := strings.SplitN(string(payload), "|", 3)
	if len(fields) != 3 {
		return nil
	}

	expiry, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return nil
	}

	return &dashboardSession{Nick: fields[0], Channel: fields[1]}
}

// dashboardAuthHandler validates a single-use auth token, sets a signed session cookie, and redirects to the dashboard.
func (s *server) dashboardAuthHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()
	tokenValue := r.PathValue("token")

	fs := firestore.Get()
	token, err := fs.GetAuthToken(tokenValue)
	if err != nil {
		logger.Errorf(nil, "error fetching auth token: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if token == nil || !token.IsValid() {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	if err := fs.MarkAuthTokenUsed(tokenValue); err != nil {
		logger.Warningf(nil, "error marking auth token as used: %s", err)
	}

	expiry := time.Now().Add(s.cfg.Web.Dashboard.SessionExpiryDuration())
	cookieValue := s.signSessionCookie(token.Nick, token.Channel, expiry)

	http.SetCookie(w, &http.Cookie{
		Name:     dashboardSessionCookie,
		Value:    cookieValue,
		Path:     "/dashboard",
		Expires:  expiry,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// dashboardHandler serves the dashboard page for authenticated users.
func (s *server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	t, err := template.ParseFiles(templatesRoot + "/dashboard.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	args := map[string]any{
		"nick":    session.Nick,
		"channel": session.Channel,
		"url":     s.cfg.Web.ExternalRootURL,
	}

	w.Header().Set("Referrer-Policy", "no-referrer")
	err = t.Execute(w, args)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
	}
}

// validateDashboardSession checks the session cookie and returns the session if valid.
func (s *server) validateDashboardSession(r *http.Request) *dashboardSession {
	cookie, err := r.Cookie(dashboardSessionCookie)
	if err != nil {
		return nil
	}
	return s.verifySessionCookie(cookie.Value)
}
