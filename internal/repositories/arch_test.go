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

// noTrackChangeAllowlist lists Set* methods that intentionally do not call
// TrackChange (keyed as "ReceiverType.MethodName").
var noTrackChangeAllowlist = map[string]bool{
	"Tag.SetManifestInfo":  true,
	"BaseModel.SetVersion": true,
}

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

// parseRepoFiles parses all non-test .go files in the repositories package
// directory (the cwd when the test runs) and returns the parsed AST files.
func (s *ChangeListArchTestSuite) parseRepoFiles() ([]*ast.File, *token.FileSet) {
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
	return files, fset
}

func (s *ChangeListArchTestSuite) TestBaseModelEmbedAlsoEmbedsChangeList() {
	files, _ := s.parseRepoFiles()

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

// TestSetMutatorsCallTrackChange verifies that every Set* method on a pointer
// receiver in the repositories package calls TrackChange somewhere in its body,
// unless explicitly listed in noTrackChangeAllowlist.
func (s *ChangeListArchTestSuite) TestSetMutatorsCallTrackChange() {
	files, fset := s.parseRepoFiles()

	for _, file := range files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			// Must have a receiver and name starting with "Set".
			if funcDecl.Recv == nil || !strings.HasPrefix(funcDecl.Name.Name, "Set") {
				continue
			}
			if funcDecl.Body == nil {
				continue
			}

			// Extract receiver type name (dereference pointer if needed).
			var receiverTypeName string
			if len(funcDecl.Recv.List) > 0 {
				recvType := funcDecl.Recv.List[0].Type
				if starExpr, ok := recvType.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						receiverTypeName = ident.Name
					}
				} else if ident, ok := recvType.(*ast.Ident); ok {
					receiverTypeName = ident.Name
				}
			}
			if receiverTypeName == "" {
				continue
			}

			key := receiverTypeName + "." + funcDecl.Name.Name
			if noTrackChangeAllowlist[key] {
				continue
			}

			// Check whether TrackChange is called anywhere in the body.
			callsTrackChange := false
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				if callsTrackChange {
					return false
				}
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := callExpr.Fun.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "TrackChange" {
					callsTrackChange = true
				}
				return true
			})

			pos := fset.Position(funcDecl.Pos())
			s.True(
				callsTrackChange,
				"%s:%d — %s does not call TrackChange; add TrackChange or add %q to noTrackChangeAllowlist",
				pos.Filename, pos.Line, key, key,
			)
		}
	}
}
