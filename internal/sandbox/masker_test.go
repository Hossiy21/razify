package sandbox_test

import (
	"bytes"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestStreamMasker_Basic(t *testing.T) {
	var buf bytes.Buffer
	secret := "my_super_secret_token_123"
	masker := sandbox.NewStreamMasker(&buf, []string{secret})

	input := "Log output: access granted with my_super_secret_token_123 to host"
	_, err := masker.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	_ = masker.Close()

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Fatalf("secret leaked in output: %s", got)
	}
	if !strings.Contains(got, sandbox.RedactedPlaceholder) {
		t.Fatalf("expected redacted placeholder, got: %s", got)
	}
}

func TestStreamMasker_ChunkedBoundary(t *testing.T) {
	var buf bytes.Buffer
	secret := "secret_api_key_456"
	masker := sandbox.NewStreamMasker(&buf, []string{secret})

	// Split secret across two writes: "secret_api_" and "key_456"
	chunk1 := "prefix log: key is secret_api_"
	chunk2 := "key_456 in request"

	_, err := masker.Write([]byte(chunk1))
	if err != nil {
		t.Fatalf("chunk1 write failed: %v", err)
	}

	_, err = masker.Write([]byte(chunk2))
	if err != nil {
		t.Fatalf("chunk2 write failed: %v", err)
	}

	_ = masker.Close()

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Fatalf("secret was not redacted across chunk boundary: %s", got)
	}
	if !strings.Contains(got, sandbox.RedactedPlaceholder) {
		t.Fatalf("expected redacted placeholder, got: %s", got)
	}
}

func TestStreamMasker_Base64AndURLEncoded(t *testing.T) {
	var buf bytes.Buffer
	secret := "token&with+special/chars"
	b64Secret := base64.StdEncoding.EncodeToString([]byte(secret))
	urlSecret := url.QueryEscape(secret)

	masker := sandbox.NewStreamMasker(&buf, []string{secret})

	input := "Raw: " + secret + "\nB64: " + b64Secret + "\nURL: " + urlSecret
	_, err := masker.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	_ = masker.Close()

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Errorf("raw secret was not redacted: %s", got)
	}
	if strings.Contains(got, b64Secret) {
		t.Errorf("base64 variant was not redacted: %s", got)
	}
	if strings.Contains(got, urlSecret) {
		t.Errorf("URL encoded variant was not redacted: %s", got)
	}
}

func TestMaskString(t *testing.T) {
	secret := "sk_test_" + "000000000000"
	input := "Bearer " + secret
	result := sandbox.MaskString(input, []string{secret})

	if strings.Contains(result, secret) {
		t.Errorf("MaskString leaked secret: %s", result)
	}
	if result != "Bearer [REDACTED]" {
		t.Errorf("unexpected MaskString result: %s", result)
	}
}

