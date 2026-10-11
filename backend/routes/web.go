package routes

import (
	"ebox/app/facades"
	"ebox/app/support/webserve"
)

func Web() {
	webserve.Register(facades.Route())
}
