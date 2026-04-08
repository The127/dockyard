package repositories_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// insertDeleteOnlyEntities are structs that embed BaseModel but intentionally
// have no mutable fields and therefore do not need change tracking via change.List.
var insertDeleteOnlyEntities = map[string]bool{
	"Blob":           true,
	"File":           true,
	"Manifest":       true,
	"Tag":            true,
	"RepositoryBlob": true,
}

type ChangeListArchTestSuite struct {
	suite.Suite
}

func TestChangeListArchTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ChangeListArchTestSuite))
}

func (s *ChangeListArchTestSuite) TestBaseModelEmbedAlsoEmbedsChangeList() {
	entries, err := os.ReadDir(".")
	s.Require().NoError(err)

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		s.Require().NoError(err)
		files = append(files, f)
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				structName := typeSpec.Name.Name
				embedsBaseModel := false
				embedsChangeList := false

				for _, field := range structType.Fields.List {
					// Embedded fields have no names
					if len(field.Names) != 0 {
						continue
					}

					switch t := field.Type.(type) {
					case *ast.Ident:
						if t.Name == "BaseModel" {
							embedsBaseModel = true
						}
					case *ast.IndexExpr:
						// change.List[T] — selector expression as the index base
						sel, ok := t.X.(*ast.SelectorExpr)
						if ok {
							pkg, ok := sel.X.(*ast.Ident)
							if ok && pkg.Name == "change" && sel.Sel.Name == "List" {
								embedsChangeList = true
							}
						}
					}
				}

				if !embedsBaseModel {
					continue
				}

				if insertDeleteOnlyEntities[structName] {
					continue
				}

				s.True(
					embedsChangeList,
					"struct %q embeds BaseModel but does not embed change.List — add change tracking or add it to insertDeleteOnlyEntities",
					structName,
				)
			}
		}
	}
}
