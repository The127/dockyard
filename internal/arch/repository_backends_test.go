package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RepositoryBackendsArchTestSuite struct {
	suite.Suite
	internalDir string
}

func TestRepositoryBackendsArchTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RepositoryBackendsArchTestSuite))
}

func (s *RepositoryBackendsArchTestSuite) SetupSuite() {
	_, thisFile, _, ok := runtime.Caller(0)
	s.Require().True(ok, "runtime.Caller failed")
	s.internalDir = filepath.Join(filepath.Dir(thisFile), "..")
}

// typeNamesInDir returns the interface and struct type names from non-test .go
// files directly under dir (non-recursive), collected in a single pass.
func (s *RepositoryBackendsArchTestSuite) typeNamesInDir(dir string) (interfaces []string, structs map[string]bool) {
	entries, err := os.ReadDir(dir)
	s.Require().NoError(err, "reading dir %s", dir)

	fset := token.NewFileSet()
	structs = make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		absPath := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, absPath, nil, 0)
		s.Require().NoError(err, "parsing %s", absPath)

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				switch typeSpec.Type.(type) {
				case *ast.InterfaceType:
					interfaces = append(interfaces, typeSpec.Name.Name)
				case *ast.StructType:
					structs[typeSpec.Name.Name] = true
				}
			}
		}
	}
	return interfaces, structs
}

// TestBothBackendsImplementEveryRepositoryInterface verifies that for every
// interface declared in internal/repositories/ there is a same-named struct in
// both internal/repositories/inmemory/ and internal/repositories/postgres/.
func (s *RepositoryBackendsArchTestSuite) TestBothBackendsImplementEveryRepositoryInterface() {
	repoDir := filepath.Join(s.internalDir, "repositories")
	inmemoryDir := filepath.Join(repoDir, "inmemory")
	postgresDir := filepath.Join(repoDir, "postgres")

	interfaces, _ := s.typeNamesInDir(repoDir)
	_, inmemoryStructs := s.typeNamesInDir(inmemoryDir)
	_, postgresStructs := s.typeNamesInDir(postgresDir)

	for _, iface := range interfaces {
		s.True(
			inmemoryStructs[iface],
			"interface %q has no matching struct in repositories/inmemory/", iface,
		)
		s.True(
			postgresStructs[iface],
			"interface %q has no matching struct in repositories/postgres/", iface,
		)
	}
}
