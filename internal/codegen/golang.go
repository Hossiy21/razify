package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// ParseGoFile parses a Go source file and extracts environment variable references using Go AST.
func ParseGoFile(filePath string, src []byte) ([]ExtractedVar, error) {
	fset := token.NewFileSet()
	var fileNode *ast.File
	var err error

	if src != nil {
		fileNode, err = parser.ParseFile(fset, filePath, src, parser.ParseComments)
	} else {
		fileNode, err = parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	}
	if err != nil {
		return nil, err
	}

	var results []ExtractedVar

	// Helper to track parent nodes
	parents := make(map[ast.Node]ast.Node)
	ast.Inspect(fileNode, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		for _, child := range getChildren(n) {
			parents[child] = n
		}
		return true
	})

	ast.Inspect(fileNode, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "os" {
			return true
		}

		fnName := sel.Sel.Name
		if fnName != "Getenv" && fnName != "LookupEnv" && fnName != "Setenv" {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		keyLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || keyLit.Kind != token.STRING {
			return true
		}

		key, err := strconv.Unquote(keyLit.Value)
		if err != nil || key == "" {
			return true
		}

		pos := fset.Position(call.Pos())
		loc := Location{
			File: filePath,
			Line: pos.Line,
		}

		// Infer type and default from AST context
		inferredType := inferTypeFromContext(key, call, parents)
		category := inferCategory(key, inferredType)
		defaultVal := inferDefaultValue(key, inferredType)
		isRequired := fnName == "LookupEnv"

		results = append(results, ExtractedVar{
			Name:         key,
			DefaultValue: defaultVal,
			InferredType: inferredType,
			IsRequired:   isRequired,
			Category:     category,
			Occurrences:  []Location{loc},
		})

		return true
	})

	return results, nil
}

func inferTypeFromContext(key string, call *ast.CallExpr, parents map[ast.Node]ast.Node) string {
	upper := strings.ToUpper(key)

	// Check AST parent expressions
	parent := parents[call]
	if parent != nil {
		if callParent, ok := parent.(*ast.CallExpr); ok {
			if sel, ok := callParent.Fun.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := sel.X.(*ast.Ident); ok && pkgIdent.Name == "strconv" {
					if sel.Sel.Name == "Atoi" || sel.Sel.Name == "ParseInt" || sel.Sel.Name == "ParseUint" {
						if strings.Contains(upper, "PORT") {
							return "port"
						}
						return "int"
					}
					if sel.Sel.Name == "ParseBool" {
						return "bool"
					}
				}
			}
		}
	}

	// Heuristics based on variable name
	if strings.Contains(upper, "PORT") {
		return "port"
	}
	if strings.Contains(upper, "URL") || strings.Contains(upper, "URI") || strings.Contains(upper, "ENDPOINT") {
		return "url"
	}
	if strings.Contains(upper, "KEY") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "AUTH") || strings.Contains(upper, "PRIVATE") {
		return "secret"
	}
	if strings.Contains(upper, "EMAIL") {
		return "email"
	}
	if strings.HasPrefix(upper, "ENABLE_") || strings.HasPrefix(upper, "DISABLE_") ||
		strings.HasPrefix(upper, "IS_") || strings.HasPrefix(upper, "HAS_") ||
		strings.Contains(upper, "DEBUG") {
		return "bool"
	}

	return "string"
}

func inferCategory(key, inferredType string) string {
	upper := strings.ToUpper(key)
	if inferredType == "port" || strings.Contains(upper, "HOST") || strings.Contains(upper, "ADDR") || strings.Contains(upper, "SERVER") {
		return "Server"
	}
	if strings.Contains(upper, "DB") || strings.Contains(upper, "DATABASE") || strings.Contains(upper, "POSTGRES") ||
		strings.Contains(upper, "MONGO") || strings.Contains(upper, "REDIS") || strings.Contains(upper, "MYSQL") {
		return "Database"
	}
	if inferredType == "secret" || strings.Contains(upper, "AUTH") || strings.Contains(upper, "JWT") || strings.Contains(upper, "SECURITY") {
		return "Security"
	}
	if inferredType == "bool" || strings.Contains(upper, "FLAG") || strings.Contains(upper, "FEATURE") || strings.Contains(upper, "DEBUG") {
		return "Feature Flags"
	}
	return "General"
}

func inferDefaultValue(key, inferredType string) string {
	upper := strings.ToUpper(key)
	switch inferredType {
	case "port":
		return "8080"
	case "url":
		if strings.Contains(upper, "DB") || strings.Contains(upper, "POSTGRES") {
			return "postgres://user:password@localhost:5432/db"
		}
		if strings.Contains(upper, "REDIS") {
			return "redis://localhost:6379"
		}
		if strings.Contains(upper, "MONGO") {
			return "mongodb://localhost:27017/db"
		}
		return "http://localhost:8080"
	case "bool":
		return "false"
	case "int":
		return "0"
	case "email":
		return "admin@example.com"
	case "secret":
		return "change_me_in_production"
	default:
		return ""
	}
}

func getChildren(n ast.Node) []ast.Node {
	var children []ast.Node
	ast.Inspect(n, func(child ast.Node) bool {
		if child != nil && child != n {
			children = append(children, child)
			return false // don't recurse here, top level inspection handles it
		}
		return true
	})
	return children
}
