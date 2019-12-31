package main

import (
	"log"

	"github.com/solo-io/anyvendor/pkg/cli"
)

func main() {
	app := cli.RootCmd()
	if err := app.Execute(); err != nil {
		log.Fatalf("error executing anyvendor cli: %+v", err)
	}
}
