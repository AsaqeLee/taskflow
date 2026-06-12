package main

import (
	"log"

	"github.com/AsaqeLee/taskflow/internal/bootstrap"
	"github.com/AsaqeLee/taskflow/internal/config"
	"github.com/AsaqeLee/taskflow/internal/observability"
)

func main() {
	cfg := config.Load()
	observability.ConfigureLogger(cfg)
	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
