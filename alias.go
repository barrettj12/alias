package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"bufio"
	"github.com/juju/gnuflag"
	"golang.org/x/tools/go/packages"
	"io/fs"
	"strings"
)

var defaultNameCache = map[string]string{}
var rootDir string
var skipGenerated bool
var skipUnique bool

func main() {
	parseArgs()

	// pkgpath -> alias -> count
	aliasInfo := map[string]map[string]int{}

	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		// Check if this is a Go source file
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Check if file is generated
		if skipGenerated {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Scan()
			fstLine := scanner.Text()

			if strings.Contains(fstLine, "generated") {
				return nil
			}
		}

		aliases, err := resolveImports(path)
		if err != nil {
			return err
		}
		for _, impt := range aliases {
			if aliasInfo[impt.path] == nil {
				aliasInfo[impt.path] = map[string]int{}
			}
			aliasInfo[impt.path][impt.alias] += 1
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	printAliasInfo(aliasInfo)
}

func parseArgs() {
	// Set flags
	gnuflag.BoolVar(&skipGenerated, "skip-generated", false, "Skip generated files")
	gnuflag.BoolVar(&skipUnique, "skip-unique", false, "Don't print packages with a unique alias")

	gnuflag.Parse(true)
	args := gnuflag.Args()

	if len(args) < 1 {
		fmt.Println(`
No dir provided.
usage:   goalias <dir>`[1:],
		)
		os.Exit(1)
	}
	rootDir = args[0]
	//fmt.Printf("rootDir: %s\n", rootDir)
	//fmt.Printf("skipGenerated: %v\n", skipGenerated)
	//fmt.Printf("skipUnique: %v\n", skipUnique)
}

type importAlias struct {
	path  string // full import path e.g. "github.com/juju/mgo/v3"
	alias string // referenced name in code e.g. "mgo"
}

// Given the path to a Go source file, get all its imports, along with their
// aliases in code.
func resolveImports(filename string) ([]importAlias, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	aliases := make([]importAlias, len(file.Imports))
	for _, impt := range file.Imports {
		name, err := getName(impt, filepath.Dir(filename))
		if err != nil {
			continue
		}
		aliases = append(aliases, importAlias{strings.Trim(impt.Path.Value, `"`), name})
	}
	return aliases, nil
}

// Get the name used to reference this import in the code.
func getName(impt *ast.ImportSpec, dir string) (string, error) {
	if impt.Name != nil {
		name := impt.Name.String()
		if name == "." || name == "_" {
			return "", fmt.Errorf("%q is not an alias", name)
		}
		return name, nil
	}

	// If no import name specified - need to find package name defined in source
	path := strings.Trim(impt.Path.Value, `"`)

	// Check cache to save time
	if name, ok := defaultNameCache[path]; ok {
		return name, nil
	}

	// Otherwise, need to load package and check name
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName, Dir: dir}, path)
	if err != nil {
		return "", err
	}
	if len(pkgs[0].Errors) > 0 {
		return "", fmt.Errorf("errors getting package %q: %v", path, pkgs[0].Errors)
	}
	defaultNameCache[path] = pkgs[0].Name
	return pkgs[0].Name, nil
}

func printAliasInfo(aliasInfo map[string]map[string]int) {
	for pkgpath, counts := range aliasInfo {
		if skipUnique && len(counts) == 1 {
			continue
		}
		fmt.Printf("Package %q has the following aliases:\n", pkgpath)
		for alias, num := range counts {
			fmt.Printf(" - %q in %d files\n", alias, num)
		}
	}
}
