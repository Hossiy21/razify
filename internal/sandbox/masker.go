package sandbox

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const RedactedPlaceholder = "[REDACTED]"

// StreamMasker wraps an io.Writer and masks known secret strings, base64 variants,
// and URL-encoded variants in a streaming fashion.
type StreamMasker struct {
	out      io.Writer
	replacer *strings.Replacer
	tail     []byte
	maxLen   int
	mu       sync.Mutex
}

// NewStreamMasker creates a StreamMasker that redacts the provided secrets and their encoded forms.
func NewStreamMasker(out io.Writer, secrets []string) *StreamMasker {
	uniqueTargets := make(map[string]bool)

	for _, s := range secrets {
		trimmed := strings.TrimSpace(s)
		if len(trimmed) < 4 {
			continue // Skip very short strings to prevent over-redaction
		}

		uniqueTargets[trimmed] = true

		// Base64 variants
		b64Std := base64.StdEncoding.EncodeToString([]byte(trimmed))
		if len(b64Std) >= 4 {
			uniqueTargets[b64Std] = true
		}
		b64Raw := base64.RawStdEncoding.EncodeToString([]byte(trimmed))
		if len(b64Raw) >= 4 {
			uniqueTargets[b64Raw] = true
		}

		// URL-encoded variant
		urlEnc := url.QueryEscape(trimmed)
		if len(urlEnc) >= 4 {
			uniqueTargets[urlEnc] = true
		}
	}

	var targetList []string
	maxLen := 0
	for t := range uniqueTargets {
		targetList = append(targetList, t)
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}

	// Sort targets by length descending
	sort.Slice(targetList, func(i, j int) bool {
		return len(targetList[i]) > len(targetList[j])
	})

	var replacePairs []string
	for _, t := range targetList {
		replacePairs = append(replacePairs, t, RedactedPlaceholder)
	}

	var replacer *strings.Replacer
	if len(replacePairs) > 0 {
		replacer = strings.NewReplacer(replacePairs...)
	}

	return &StreamMasker{
		out:      out,
		replacer: replacer,
		maxLen:   maxLen,
	}
}

// Write processes a chunk of output, redacting secrets and preserving boundary state.
func (sm *StreamMasker) Write(p []byte) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.replacer == nil || sm.maxLen <= 0 {
		return sm.out.Write(p)
	}

	combined := append(sm.tail, p...)
	masked := sm.replacer.Replace(string(combined))
	maskedBytes := []byte(masked)

	// Keep up to maxLen - 1 bytes in tail for cross-chunk matches
	keepLen := sm.maxLen - 1
	if keepLen > len(maskedBytes) {
		keepLen = len(maskedBytes)
	}

	splitIdx := len(maskedBytes) - keepLen
	toWrite := maskedBytes[:splitIdx]
	sm.tail = append([]byte(nil), maskedBytes[splitIdx:]...)

	if len(toWrite) > 0 {
		if _, err := sm.out.Write(toWrite); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Close flushes any remaining buffered bytes in the tail buffer.
func (sm *StreamMasker) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.tail) > 0 {
		var toWrite []byte
		if sm.replacer != nil {
			masked := sm.replacer.Replace(string(sm.tail))
			toWrite = []byte(masked)
		} else {
			toWrite = sm.tail
		}
		sm.tail = nil
		_, err := sm.out.Write(toWrite)
		return err
	}
	return nil
}

// MaskString applies the secret redaction rules directly to an in-memory string.
func MaskString(input string, secrets []string) string {
	var buf bytes.Buffer
	masker := NewStreamMasker(&buf, secrets)
	_, _ = masker.Write([]byte(input))
	_ = masker.Close()
	return buf.String()
}

