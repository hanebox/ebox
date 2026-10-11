package httpx

import (
	"net/http"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
)

const (
	SessionCookie = "ebox_session"
	CSRFCookie    = "ebox_csrf"

	ContextSession = "ebox:session"
	ContextUser    = "ebox:user"
)

func Ok(ctx contractshttp.Context, data any) contractshttp.Response {
	return ctx.Response().Json(http.StatusOK, data)
}

func Created(ctx contractshttp.Context, data any) contractshttp.Response {
	return ctx.Response().Json(http.StatusCreated, data)
}

func Err(ctx contractshttp.Context, status int, message string) contractshttp.Response {
	return ctx.Response().Status(status).Json(contractshttp.Json{"message": message})
}

func ValidationErrors(ctx contractshttp.Context, errors map[string][]string) contractshttp.Response {
	return ctx.Response().Status(http.StatusUnprocessableEntity).Json(contractshttp.Json{
		"message": "The given data was invalid.",
		"errors":  errors,
	})
}

func SetSessionCookies(ctx contractshttp.Context, sessionToken, csrfToken string, secure bool, expiresAt time.Time) {
	ctx.Response().Cookie(contractshttp.Cookie{
		Name:     SessionCookie,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   secure,
		HttpOnly: true,
		SameSite: "lax",
	})
	ctx.Response().Cookie(contractshttp.Cookie{
		Name:     CSRFCookie,
		Value:    csrfToken,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   secure,
		HttpOnly: false,
		SameSite: "lax",
	})
}

func ClearSessionCookies(ctx contractshttp.Context) {
	ctx.Response().WithoutCookie(SessionCookie)
	ctx.Response().WithoutCookie(CSRFCookie)
}

func Session(ctx contractshttp.Context) *models.Session {
	session, ok := ctx.Value(ContextSession).(*models.Session)
	if !ok {
		return nil
	}

	return session
}

func User(ctx contractshttp.Context) *models.User {
	user, ok := ctx.Value(ContextUser).(*models.User)
	if !ok {
		return nil
	}

	return user
}
