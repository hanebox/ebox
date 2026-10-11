package middleware

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/account"
	"ebox/app/support/httpx"
)

type Authenticate struct{}

func (m *Authenticate) Signature() string {
	return "ebox:authenticate"
}

func (m *Authenticate) Handle(ctx contractshttp.Context) {
	if header := ctx.Request().Header("Authorization"); strings.HasPrefix(header, "Bearer ") {
		m.authenticateAccessToken(ctx, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		return
	}

	token := ctx.Request().Cookie(httpx.SessionCookie)
	if token == "" {
		m.reject(ctx)
		return
	}

	session, user, err := account.Default().ValidateSession(token)
	if err != nil {
		httpx.ClearSessionCookies(ctx)
		m.reject(ctx)
		return
	}

	ctx.WithValue(httpx.ContextSession, session)
	ctx.WithValue(httpx.ContextUser, user)

	ctx.Request().Next()
}

func (m *Authenticate) authenticateAccessToken(ctx contractshttp.Context, raw string) {
	if raw == "" {
		m.reject(ctx)
		return
	}

	record, user, err := account.Default().AuthenticateAccessToken(raw)
	if err != nil {
		m.reject(ctx)
		return
	}

	account.Default().TouchAccessToken(record)

	ctx.WithValue(httpx.ContextAccessToken, record)
	ctx.WithValue(httpx.ContextUser, user)

	ctx.Request().Next()
}

func (m *Authenticate) reject(ctx contractshttp.Context) {
	ctx.Request().AbortWithStatusJson(http.StatusUnauthorized, contractshttp.Json{"message": "Unauthenticated."})
}
