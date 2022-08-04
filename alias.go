package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/gosuri/uitable"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Println(`
No filename provided.
usage:   alias <filename>`[1:],
		)
		os.Exit(1)
	}
	filename := os.Args[1]

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		panic(err)
	}

	table := uitable.New()
	for _, impt := range file.Imports {
		table.AddRow(getName(impt), impt.Path.Value)
		//fmt.Println(impt)
	}
	fmt.Println(table)
}

func getName(impt *ast.ImportSpec) string {
	if impt.Name == nil {
		unquoted := strings.Trim(impt.Path.Value, `"`)
		base := filepath.Base(unquoted)
		return strings.Split(base, ".")[0]
	}
	return impt.Name.String()
}
