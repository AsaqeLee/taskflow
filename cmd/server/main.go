package main

import (
	"log"

	"github.com/AsaqeLee/taskflow/internal/bootstrap"
	"github.com/AsaqeLee/taskflow/internal/config"
)

func main() {
	cfg := config.Load()
	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
