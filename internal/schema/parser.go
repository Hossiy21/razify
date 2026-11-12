package schema

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	tagParenRegex      = regexp.MustCompile(`@(\w+)\(([^)]+)\)`)
	tagEqRegex         = regexp.MustCompile(`@(\w+)=([\w\-\.,]+)`)
	tagStandaloneRegex = regexp.MustCompile(`@(\w+)\b`)
	tagStripRegex      = regexp.MustCompile(`@\w+=[\w\-\.,]+`)
)

// ParseErrorCode identifies the category of a syntax or formatting error in an env file.
type ParseErrorCode string

const (
	ErrMissingEquals     ParseErrorCode = "MISSING_EQUALS"
	ErrInvalidKey        ParseErrorCode = "INVALID_KEY"
	ErrUnterminatedQuote ParseErrorCode = "UNTERMINATED_QUOTE"
	ErrLineTooLong       ParseErrorCode = "LINE_TOO_LONG"
	ErrSyntaxError       ParseErrorCode = "SYNTAX_ERROR"
)

// ParseError represents a structured dotenv parsing failure with NO raw input text.
type ParseError struct {
	Line int            `json:"line"`
	Code ParseErrorCode `json:"code"`
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at line %d: %s", e.Line, e.Code)
}

// ParseEnvFile parses a basic key=value .env file.
func ParseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseEnvReader(file)
}

// ParseEnvReader parses key=value pairs from an io.Reader.
func ParseEnvReader(r io.Reader) (map[string]string, error) {
	vars, err := ParseEnvWithMetadataReader(r)
	if err != nil {
		return nil, err
	}
	res := make(map[string]string, len(vars))
	for _, v := range vars {
		res[v.Key] = v.Value
	}
	return res, nil
}

// ParseEnvWithMetadata parses a .env or .env.example file extracting comments, categories, and @tags.
func ParseEnvWithMetadata(filename string) ([]EnvVar, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseEnvWithMetadataReader(file)
}

// ParseEnvWithMetadataReader parses .env file lines with comments and schema metadata from an io.Reader.
func ParseEnvWithMetadataReader(r io.Reader) ([]EnvVar, error) {
	var vars []EnvVar
	var lastComment string
	var currentCategory string
	lineNum := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			lastComment = ""
			continue
		}

		if strings.HasPrefix(line, "##") {
			currentCategory = strings.TrimSpace(strings.TrimPrefix(line, "##"))
			lastComment = ""
			continue
		}

		if strings.HasPrefix(line, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, &ParseError{Line: lineNum, Code: ErrMissingEquals}
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, &ParseError{Line: lineNum, Code: ErrInvalidKey}
		}

		valueWithComment := strings.TrimSpace(parts[1])
		value := valueWithComment
		trailingComment := ""

		if strings.HasPrefix(valueWithComment, "\"") || strings.HasPrefix(valueWithComment, "'") {
			quoteChar := valueWithComment[0]
			startLine := lineNum

			// Check if quote closes on this line
			closeIdx := -1
			for i := 1; i < len(valueWithComment); i++ {
				if valueWithComment[i] == quoteChar && valueWithComment[i-1] != '\\' {
					closeIdx = i
					break
				}
			}

			if closeIdx != -1 {
				value = valueWithComment[1:closeIdx]
				rest := strings.TrimSpace(valueWithComment[closeIdx+1:])
				if strings.HasPrefix(rest, "#") {
					trailingComment = strings.TrimSpace(strings.TrimPrefix(rest, "#"))
				}
			} else {
				// Multi-line quoted value
				var sb strings.Builder
				sb.WriteString(valueWithComment[1:])
				closed := false

				for scanner.Scan() {
					lineNum++
					nextLine := scanner.Text()
					foundIdx := -1
					for i := 0; i < len(nextLine); i++ {
						if nextLine[i] == quoteChar && (i == 0 || nextLine[i-1] != '\\') {
							foundIdx = i
							break
						}
					}

					if foundIdx != -1 {
						sb.WriteString("\n")
						sb.WriteString(nextLine[:foundIdx])
						rest := strings.TrimSpace(nextLine[foundIdx+1:])
						if strings.HasPrefix(rest, "#") {
							trailingComment = strings.TrimSpace(strings.TrimPrefix(rest, "#"))
						}
						closed = true
						break
					} else {
						sb.WriteString("\n")
						sb.WriteString(nextLine)
					}
				}

				if !closed {
					return nil, &ParseError{Line: startLine, Code: ErrUnterminatedQuote}
				}
				value = sb.String()
			}
		} else {
			// Unquoted value: check for trailing comment
			if hashIdx := strings.Index(valueWithComment, "#"); hashIdx != -1 {
				value = strings.TrimSpace(valueWithComment[:hashIdx])
				trailingComment = strings.TrimSpace(valueWithComment[hashIdx+1:])
			} else {
				value = valueWithComment
			}
		}

		commentToParse := lastComment
		if trailingComment != "" {
			if commentToParse != "" {
				commentToParse += " " + trailingComment
			} else {
				commentToParse = trailingComment
			}
		}

		tags := make(map[string]string)
		if commentToParse != "" {
			// 1. Parentheses syntax: @tag(val) e.g. @type(url), @enum(dev,prod), @requires(DB_HOST)
			for _, m := range tagParenRegex.FindAllStringSubmatch(commentToParse, -1) {
				tags[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}

			// 2. Equals syntax: @tag=val e.g. @type=int, @range=1-100
			for _, m := range tagEqRegex.FindAllStringSubmatch(commentToParse, -1) {
				tags[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}

			// 3. Standalone boolean flags e.g. @required
			for _, m := range tagStandaloneRegex.FindAllStringSubmatch(commentToParse, -1) {
				tagName := strings.ToLower(m[1])
				if _, exists := tags[tagName]; !exists {
					tags[tagName] = "true"
				}
			}
		}

		// Required if value is empty or has a "your-" prefix, or if @required tag exists
		required := value == "" || strings.HasPrefix(value, "your-")
		if val, ok := tags["required"]; ok {
			required = val == "true" || val == "yes" || val == "1"
		}

		vars = append(vars, EnvVar{
			Key:      key,
			Value:    value,
			Comment:  lastComment,
			Required: required,
			Category: currentCategory,
			Tags:     tags,
			LineNum:  lineNum,
			Ignored:  strings.Contains(lastComment, "razify:ignore") || strings.Contains(trailingComment, "razify:ignore"),
		})

		lastComment = ""
	}

	if err := scanner.Err(); err != nil {
		return nil, &ParseError{Line: lineNum, Code: ErrLineTooLong}
	}

	return vars, nil
}

// StripTags removes @tag=value substrings from a comment string.
func StripTags(comment string) string {
	return strings.TrimSpace(tagStripRegex.ReplaceAllString(comment, ""))
}
