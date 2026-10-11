package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/services/account"
	"ebox/app/services/crypto"
	"ebox/app/support/httpx"
)

type AuthController struct{}

func NewAuthController() *AuthController {
	return &AuthController{}
}

func (c *AuthController) Login(ctx http.Context) http.Response {
	email := strings.TrimSpace(ctx.Request().Input("email"))
	password := ctx.Request().Input("password")

	errors := newValidationErrors()
	if email == "" {
		errors.add("email", "The email field is required.")
	}
	if password == "" {
		errors.add("password", "The password field is required.")
	}
	if !errors.empty() {
		return httpx.ValidationErrors(ctx, errors)
	}

	limiter := account.LoginLimiter()
	limiterKey := ctx.Request().Ip()
	if !limiter.Allow(limiterKey) {
		return httpx.Err(ctx, http.StatusTooManyRequests, "Too many failed attempts. Please try again later.")
	}

	accountService := account.Default()
	user, err := accountService.Authenticate(email, password)
	if err != nil {
		limiter.RecordFailure(limiterKey)
		return httpx.Err(ctx, http.StatusUnauthorized, "These credentials do not match our records.")
	}
	limiter.Reset(limiterKey)

	session, token, csrf, err := accountService.CreateSession(user, ctx.Request().Ip(), ctx.Request().Header("User-Agent"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create a session.")
	}

	return respondWithSession(ctx, user, session.ExpiresAt, token, csrf)
}

func (c *AuthController) Me(ctx http.Context) http.Response {
	user := httpx.User(ctx)
	session := httpx.Session(ctx)
	if user == nil || session == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	csrf := ctx.Request().Cookie(httpx.CSRFCookie)
	if csrf == "" {
		rotated, err := crypto.RandomToken(32)
		if err != nil {
			return httpx.Err(ctx, http.StatusInternalServerError, "Failed to refresh the CSRF token.")
		}

		session.CSRFHash = crypto.HashToken(rotated)
		if err := facades.Orm().Query().Save(session); err != nil {
			return httpx.Err(ctx, http.StatusInternalServerError, "Failed to refresh the CSRF token.")
		}

		httpx.SetSessionCookies(ctx, ctx.Request().Cookie(httpx.SessionCookie), rotated, cookieSecure(ctx), session.ExpiresAt)
		csrf = rotated
	}

	return httpx.Ok(ctx, http.Json{
		"user":       user,
		"csrf_token": csrf,
	})
}

func (c *AuthController) Logout(ctx http.Context) http.Response {
	session := httpx.Session(ctx)
	if session != nil {
		_ = account.Default().RevokeSession(session.UserID, session.ID)
	}

	httpx.ClearSessionCookies(ctx)

	return httpx.Ok(ctx, http.Json{"message": "Logged out."})
}
