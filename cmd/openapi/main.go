package main

import (
	"encoding/json"
	"fmt"
	"github.com/Hasras-code/PMT_WEB.git/internal/httpapi"
	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"os"
)

func main() {
	a := &httpapi.API{}
	doc, e := openapi.Generate(a.Router())
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if e = encoder.Encode(doc); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
