package middleware

import (
	"crypto/subtle"
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/crypto"
	"ebox/app/support/httpx"
)

type Csrf struct{}

func (m *Csrf) Signature() string {
	return "ebox:csrf"
}

func (m *Csrf) Handle(ctx contractshttp.Context) {
	if httpx.AccessToken(ctx) != nil {
		ctx.Request().Next()
		return
	}

	method := ctx.Request().Method()
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		ctx.Request().Next()
		return
	}

	token := ctx.Request().Header("X-CSRF-Token")
	session := httpx.Session(ctx)
	if session == nil || token == "" || subtle.ConstantTimeCompare([]byte(crypto.HashToken(token)), []byte(session.CSRFHash)) != 1 {
		ctx.Request().AbortWithStatusJson(http.StatusForbidden, contractshttp.Json{"message": "Invalid CSRF token."})
		return
	}

	ctx.Request().Next()
}
