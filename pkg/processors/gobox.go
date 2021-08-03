package processors

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/pkg/errors"
)

// GoBox implements the processors.Processor interface to facilitate the migration
// from the go-outreach repository as a dependency to gobox.
type GoBox struct{}

// Register returns the Config for the GoBox proecssor.
func (g *GoBox) Register() *Config {
	return &Config{
		FileExtensions:         []string{".go"},
		IsPostCodegenProcessor: true,
	}
}

// Process runs the GoBox processor.
func (g *GoBox) Process(orig, _ *File) (*File, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, orig.Name, orig.Reader, parser.ParseComments)
	if err != nil {
		return nil, errors.Wrap(err, "parse file")
	}

	var newImports []*ast.ImportSpec

	ast.Inspect(f, func(n ast.Node) bool {
		if x, ok := n.(*ast.ImportSpec); ok {
			if strings.HasPrefix(x.Path.Value, `"github.com/getoutreach/go-outreach/v2`) {
				x.Path.Value = strings.Replace(x.Path.Value, "github.com/getoutreach/go-outreach/v2", "github.com/getoutreach/gobox", 1)
			}

			newImports = append(newImports, x)
		}

		return true
	})

	f.Imports = newImports

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, f); err != nil {
		return nil, errors.Wrap(err, "write ast to file")
	}

	return &File{
		Name:   orig.Name,
		Reader: &buf,
	}, nil
}
