package main

import (
	"encoding/json"
	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRouteContract(t *testing.T) {
	doc, e := openapi.Generate((&app{}).mount())
	if e != nil {
		t.Fatal(e)
	}
	paths := doc["paths"].(openapi.M)
	if len(paths) < 90 {
		t.Fatalf("unexpected route count %d", len(paths))
	}
	for path := range paths {
		if strings.Contains(path, "achievement") || strings.Contains(path, "album") || strings.Contains(path, "/batches/{batchID}/gallery") {
			t.Fatal("out-of-scope route", path)
		}
	}
	raw, e := os.ReadFile("../../docs/openapi.json")
	if e != nil {
		t.Fatal(e)
	}
	var saved, generated any
	if e = json.Unmarshal(raw, &saved); e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &generated); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(saved, generated) {
		t.Fatal("OpenAPI drift: run make docs")
	}
}
