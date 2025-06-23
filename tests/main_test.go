package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	err := os.Chdir("..")
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
