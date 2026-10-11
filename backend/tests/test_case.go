package tests

import (
	"github.com/goravel/framework/testing"

	"ebox/bootstrap"
)

func init() {
	bootstrap.Boot()
}

type TestCase struct {
	testing.TestCase
}
