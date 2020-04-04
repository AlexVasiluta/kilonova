package api

import (
	"context"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
)

func (s *API) mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.IsAuthed(r) {
			http.Error(w, "You must not be logged in to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *API) mustBeAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.IsAdmin(r) {
			http.Error(w, "You must be an admin to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *API) mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if !s.IsAuthed(r) {
			http.Error(w, "You must be authenticated to view this", http.StatusUnauthorized)
			return
		}
		var user models.User
		session := s.GetSessionCookie(r)
		s.db.First(&user, "id = ?", session.UserID)
		ctx = context.WithValue(ctx, models.KNContextType("user"), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsAuthed reads the session cookie and says if the requester is authenticated
func (s *API) IsAuthed(r *http.Request) bool {
	session := s.GetSessionCookie(r)
	if session == nil {
		return false
	}
	return session.UserID != 0
}

// IsAdmin reads the session cookie and says if the requester is an admin
func (s *API) IsAdmin(r *http.Request) bool {
	session := s.GetSessionCookie(r)
	if session == nil {
		return false
	}
	return session.IsAdmin
}
