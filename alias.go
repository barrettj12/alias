package main

import (
	"fmt"
	"github.com/gosuri/uitable"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
	"strings"
)

func main() {
	if len(os.Args) <= 2 {
		fmt.Println(`
No filename provided.
usage:   alias -- <filename>`[1:],
		)
		os.Exit(1)
	}
	filename := os.Args[2]

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		panic(err)
	}

	table := uitable.New()
	for _, impt := range file.Imports {
		name, err := getName(impt, filepath.Dir(filename))
		if err != nil {
			panic(err)
		}
		table.AddRow(name, impt.Path.Value)
	}
	fmt.Println(table)
}

func getName(impt *ast.ImportSpec, dir string) (string, error) {
	if impt.Name != nil {
		return impt.Name.String(), nil
	}

	// If no import name specified - need to find package name defined in source
	path := strings.Trim(impt.Path.Value, `"`)
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName, Dir: dir}, path)
	if err != nil {
		return "", err
	}
	if len(pkgs[0].Errors) > 0 {
		return "", fmt.Errorf("errors getting package %q: %v", path, pkgs[0].Errors)
	}
	return pkgs[0].Name, nil
}
