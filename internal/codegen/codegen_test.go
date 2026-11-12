package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/schema"
)

func TestParseGoFile(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	portStr := os.Getenv("PORT")
	port, _ := strconv.Atoi(portStr)
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		panic("missing db")
	}
	debug := os.Getenv("DEBUG")
	stripeKey := os.Getenv("STRIPE_KEY")

	fmt.Println(port, dbURL, debug, stripeKey)
}
`
	vars, err := ParseGoFile("main.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseGoFile failed: %v", err)
	}

	varMap := make(map[string]ExtractedVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}

	if v, ok := varMap["PORT"]; !ok || v.InferredType != "port" {
		t.Errorf("expected PORT type port, got %+v", v)
	}
	if v, ok := varMap["DATABASE_URL"]; !ok || v.InferredType != "url" || !v.IsRequired {
		t.Errorf("expected DATABASE_URL type url and required, got %+v", v)
	}
	if v, ok := varMap["DEBUG"]; !ok || v.InferredType != "bool" {
		t.Errorf("expected DEBUG type bool, got %+v", v)
	}
	if v, ok := varMap["STRIPE_KEY"]; !ok || v.InferredType != "secret" {
		t.Errorf("expected STRIPE_KEY type secret, got %+v", v)
	}
}

func TestParseJSFile(t *testing.T) {
	src := `
import express from 'express';

const port = parseInt(process.env.PORT || '3000', 10);
const dbUrl = process.env['DATABASE_URL'];
const isDebug = process.env.DEBUG === 'true';
const viteApi = import.meta.env.VITE_API_URL || 'http://localhost:4000';
`
	vars, err := ParseJSFile("server.ts", []byte(src))
	if err != nil {
		t.Fatalf("ParseJSFile failed: %v", err)
	}

	varMap := make(map[string]ExtractedVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}

	if v, ok := varMap["PORT"]; !ok || v.InferredType != "port" || v.DefaultValue != "3000" {
		t.Errorf("expected PORT type port default 3000, got %+v", v)
	}
	if v, ok := varMap["DATABASE_URL"]; !ok || v.InferredType != "url" {
		t.Errorf("expected DATABASE_URL type url, got %+v", v)
	}
	if v, ok := varMap["DEBUG"]; !ok || v.InferredType != "bool" {
		t.Errorf("expected DEBUG type bool, got %+v", v)
	}
	if v, ok := varMap["VITE_API_URL"]; !ok || v.InferredType != "url" || v.DefaultValue != "http://localhost:4000" {
		t.Errorf("expected VITE_API_URL type url, got %+v", v)
	}
}

func TestParsePythonFile(t *testing.T) {
	src := `
import os

port = int(os.getenv("PORT", "8000"))
db_url = os.environ.get("DATABASE_URL", "postgres://localhost:5432/db")
secret = os.environ["SECRET_KEY"]
debug = os.getenv("DEBUG", "false").lower() == "true"
`
	vars, err := ParsePythonFile("app.py", []byte(src))
	if err != nil {
		t.Fatalf("ParsePythonFile failed: %v", err)
	}

	varMap := make(map[string]ExtractedVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}

	if v, ok := varMap["PORT"]; !ok || v.InferredType != "port" || v.DefaultValue != "8000" {
		t.Errorf("expected PORT type port default 8000, got %+v", v)
	}
	if v, ok := varMap["DATABASE_URL"]; !ok || v.InferredType != "url" {
		t.Errorf("expected DATABASE_URL type url, got %+v", v)
	}
	if v, ok := varMap["SECRET_KEY"]; !ok || v.InferredType != "secret" || !v.IsRequired {
		t.Errorf("expected SECRET_KEY type secret and required, got %+v", v)
	}
	if v, ok := varMap["DEBUG"]; !ok || v.InferredType != "bool" || v.DefaultValue != "false" {
		t.Errorf("expected DEBUG type bool default false, got %+v", v)
	}
}

func TestExtract_MultiLanguageAndDeduplication(t *testing.T) {
	tempDir := t.TempDir()

	goSrc := `package main
import "os"
func run() {
	_ = os.Getenv("PORT")
	_ = os.Getenv("APP_ENV")
}
`
	jsSrc := `
const port = process.env.PORT || '9000';
const dbUrl = process.env.DATABASE_URL;
`
	pySrc := `
import os
secret = os.environ["API_SECRET"]
`
	_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(goSrc), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "index.ts"), []byte(jsSrc), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "worker.py"), []byte(pySrc), 0644)

	report, err := Extract(ExtractOptions{RootDir: tempDir})
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if report.FilesScanned != 3 {
		t.Errorf("expected 3 files scanned, got %d", report.FilesScanned)
	}

	if len(report.LanguagesDetected) < 3 {
		t.Errorf("expected 3 languages detected, got %+v", report.LanguagesDetected)
	}

	varMap := make(map[string]ExtractedVar)
	for _, v := range report.Variables {
		varMap[v.Name] = v
	}

	// PORT should be merged across Go and TS
	if port, ok := varMap["PORT"]; !ok {
		t.Errorf("expected PORT to be extracted")
	} else {
		if len(port.Occurrences) != 2 {
			t.Errorf("expected 2 occurrences for PORT, got %d", len(port.Occurrences))
		}
		if port.DefaultValue != "9000" {
			t.Errorf("expected merged default 9000, got %s", port.DefaultValue)
		}
	}

	// Generated .env.example should be valid
	template := GenerateEnvExample(report.Variables)
	if !strings.Contains(template, "PORT=9000") || !strings.Contains(template, "API_SECRET=") {
		t.Fatalf("generated template missing expected content:\n%s", template)
	}

	// Verify template parses cleanly with schema parser
	envVars, err := schema.ParseEnvWithMetadataReader(strings.NewReader(template))
	if err != nil {
		t.Fatalf("failed to parse generated template with schema parser: %v", err)
	}

	if len(envVars) != 4 {
		t.Errorf("expected 4 variables parsed from generated template, got %d", len(envVars))
	}

	// Validate each value against schema rules
	envMap := make(map[string]string)
	for _, ev := range envVars {
		envMap[ev.Key] = ev.Value
	}

	for _, ev := range envVars {
		if errMsg := schema.ValidateValue(ev.Value, ev.Tags, envMap); errMsg != "" {
			t.Errorf("validation error for generated var %s: %s", ev.Key, errMsg)
		}
	}
}

// Benchmarks

func BenchmarkExtract_Go(b *testing.B) {
	src := []byte(`package main
import "os"
func test() {
	_ = os.Getenv("PORT")
	_ = os.Getenv("DATABASE_URL")
	_ = os.Getenv("SECRET_KEY")
}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseGoFile("main.go", src)
	}
}

func BenchmarkExtract_JS(b *testing.B) {
	src := []byte(`
const port = process.env.PORT || '8080';
const db = process.env.DATABASE_URL;
const key = process.env.SECRET_KEY;
`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseJSFile("app.js", src)
	}
}

func BenchmarkExtract_Python(b *testing.B) {
	src := []byte(`
import os
port = os.getenv("PORT", "8080")
db = os.getenv("DATABASE_URL")
key = os.environ["SECRET_KEY"]
`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParsePythonFile("app.py", src)
	}
}

func BenchmarkGenerateEnvExample(b *testing.B) {
	vars := []ExtractedVar{
		{Name: "PORT", InferredType: "port", DefaultValue: "8080", Category: "Server"},
		{Name: "DATABASE_URL", InferredType: "url", DefaultValue: "postgres://localhost:5432/db", Category: "Database"},
		{Name: "SECRET_KEY", InferredType: "secret", IsRequired: true, Category: "Security"},
		{Name: "DEBUG", InferredType: "bool", DefaultValue: "false", Category: "Feature Flags"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateEnvExample(vars)
	}
}
