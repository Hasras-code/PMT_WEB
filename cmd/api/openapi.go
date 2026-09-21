package main

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/config"
)

func writeOpenAPI(w io.Writer) error {
	a := &app{
		cfg:    config.Config{Env: "development"},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	document, err := openapi.Generate(a.mount())
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}
