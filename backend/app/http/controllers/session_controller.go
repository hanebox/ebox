package controllers

import (
	"time"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/services/account"
	"ebox/app/support/httpx"
)

type SessionController struct{}

func NewSessionController() *SessionController {
	return &SessionController{}
}

type sessionResource struct {
	ID         uint64    `json:"id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

func (c *SessionController) Index(ctx http.Context) http.Response {
	user := httpx.User(ctx)
	current := httpx.Session(ctx)
	if user == nil || current == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	sessions, err := account.Default().ListSessions(user.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load sessions.")
	}

	resources := make([]sessionResource, 0, len(sessions))
	for _, session := range sessions {
		resources = append(resources, sessionResource{
			ID:         session.ID,
			IP:         session.IP,
			UserAgent:  session.UserAgent,
			LastSeenAt: session.LastSeenAt,
			CreatedAt:  session.CreatedAt,
			ExpiresAt:  session.ExpiresAt,
			Current:    session.ID == current.ID,
		})
	}

	return httpx.Ok(ctx, resources)
}

func (c *SessionController) Destroy(ctx http.Context) http.Response {
	user := httpx.User(ctx)
	current := httpx.Session(ctx)
	if user == nil || current == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	sessionID := uint64(ctx.Request().RouteInt("id"))
	if sessionID == 0 {
		return httpx.Err(ctx, http.StatusBadRequest, "Invalid session id.")
	}

	if err := account.Default().RevokeSession(user.ID, sessionID); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to revoke the session.")
	}

	if sessionID == current.ID {
		httpx.ClearSessionCookies(ctx)
	}

	return httpx.Ok(ctx, http.Json{"message": "Session revoked."})
}
