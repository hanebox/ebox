package main

import (
	"ebox/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	app.Start()
}
