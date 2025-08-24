package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <test-directory>\n", os.Args[0])
		os.Exit(1)
	}

	testDir := os.Args[1]
	tests, err := discoverTests(testDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering tests: %v\n", err)
		os.Exit(1)
	}

	for _, test := range tests {
		fmt.Println(test)
	}
}

func discoverTests(dir string) ([]string, error) {
	var tests []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-test files
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip testhelpers directory
		if strings.Contains(path, "testhelpers/") {
			return nil
		}

		// Parse the Go file
		fileTests, err := getTestsFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		tests = append(tests, fileTests...)
		return nil
	})

	return tests, err
}

func getTestsFromFile(filename string) ([]string, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, err
	}

	var tests []string
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "Test") && fn.Recv == nil {
				// Check if it's a proper test function (takes *testing.T)
				if len(fn.Type.Params.List) == 1 {
					if starExpr, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr); ok {
						if sel, ok := starExpr.X.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == "T" {
								tests = append(tests, fn.Name.Name)
							}
						}
					}
				}
			}
		}
	}

	return tests, nil
}