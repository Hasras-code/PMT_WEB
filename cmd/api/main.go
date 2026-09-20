package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx := context.Background()
	app, err := newApp(ctx, log)
	if err != nil {
		return err
	}
	defer app.close()
	return app.run()
}
