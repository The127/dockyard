package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// handlerDatabaseRepoAllowlist contains handler files that are temporarily
// allowed to import repositories or database packages while ocihandlers are
// being migrated to CQRS. Remove entries here once the files are cleaned up.
var handlerDatabaseRepoAllowlist = map[string]bool{
	"handlers/ocihandlers/blobs.go":     true,
	"handlers/ocihandlers/manifests.go": true,
	"handlers/ocihandlers/tokens.go":    true,
	"handlers/ocihandlers/utils.go":     true,
}

type DependencyArchTestSuite struct {
	suite.Suite
	internalDir string
}

func TestDependencyArchTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DependencyArchTestSuite))
}

func (s *DependencyArchTestSuite) SetupSuite() {
	_, thisFile, _, ok := runtime.Caller(0)
	s.Require().True(ok, "runtime.Caller failed")
	s.internalDir = filepath.Join(filepath.Dir(thisFile), "..")
}

// goFilesUnder recursively collects all non-test .go files under dir.
// It returns a map from relative-to-internalDir path -> absolute path.
func (s *DependencyArchTestSuite) goFilesUnder(dir string) map[string]string {
	result := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(s.internalDir, path)
		if relErr != nil {
			return relErr
		}
		result[rel] = path
		return nil
	})
	s.Require().NoError(err)
	return result
}

// importsOf returns the import paths declared in a single .go file.
func (s *DependencyArchTestSuite) importsOf(absPath string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, parser.ImportsOnly)
	s.Require().NoError(err, "parsing %s", absPath)

	imports := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		s.Require().NoError(err, "unquoting import path %s", imp.Path.Value)
		imports = append(imports, path)
	}
	return imports
}

// assertNoImportsMatching walks all non-test .go files under each of dirs and
// asserts that none of them import a path that starts with any entry in
// forbidden. Files whose relative path (from internalDir) appears in allowlist
// are skipped.
func (s *DependencyArchTestSuite) assertNoImportsMatching(dirs []string, forbidden []string, allowlist map[string]bool) {
	for _, dir := range dirs {
		files := s.goFilesUnder(dir)
		for relPath, absPath := range files {
			if allowlist[relPath] {
				continue
			}
			for _, imp := range s.importsOf(absPath) {
				for _, prefix := range forbidden {
					s.False(
						strings.HasPrefix(imp, prefix),
						"file %q must not import %q, but imports %q",
						relPath,
						prefix,
						imp,
					)
				}
			}
		}
	}
}

// TestHandlersMustNotImportRepositoriesOrDatabase verifies that no file under
// internal/handlers/ imports the repositories or database packages directly.
func (s *DependencyArchTestSuite) TestHandlersMustNotImportRepositoriesOrDatabase() {
	s.assertNoImportsMatching(
		[]string{filepath.Join(s.internalDir, "handlers")},
		[]string{
			"github.com/the127/dockyard/internal/repositories",
			"github.com/the127/dockyard/internal/database",
		},
		handlerDatabaseRepoAllowlist,
	)
}

// TestCommandsAndQueriesMustNotImportHandlers verifies that no file under
// internal/commands/ or internal/queries/ imports the handlers package.
func (s *DependencyArchTestSuite) TestCommandsAndQueriesMustNotImportHandlers() {
	s.assertNoImportsMatching(
		[]string{
			filepath.Join(s.internalDir, "commands"),
			filepath.Join(s.internalDir, "queries"),
		},
		[]string{"github.com/the127/dockyard/internal/handlers"},
		nil,
	)
}

// handleFuncsInDir returns all top-level function names starting with "Handle"
// in non-test .go files directly under dir (non-recursive), keyed as "pkg.Name".
func (s *DependencyArchTestSuite) handleFuncsInDir(dir string, pkg string) map[string]bool {
	entries, err := os.ReadDir(dir)
	s.Require().NoError(err, "reading dir %s", dir)

	result := make(map[string]bool, len(entries))
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		absPath := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, absPath, nil, 0)
		s.Require().NoError(err, "parsing %s", absPath)

		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv != nil {
				continue
			}
			if strings.HasPrefix(funcDecl.Name.Name, "Handle") {
				result[pkg+"."+funcDecl.Name.Name] = true
			}
		}
	}
	return result
}

// TestAllHandlersRegisteredWithMediator verifies that every Handle* function in
// internal/commands/ and internal/queries/ is registered via
// mediatr.RegisterHandler in internal/setup/mediator.go.
func (s *DependencyArchTestSuite) TestAllHandlersRegisteredWithMediator() {
	commandsDir := filepath.Join(s.internalDir, "commands")
	queriesDir := filepath.Join(s.internalDir, "queries")
	mediatorFile := filepath.Join(s.internalDir, "setup", "mediator.go")

	// Collect all Handle* functions declared in commands and queries.
	declared := make(map[string]bool)
	maps.Copy(declared, s.handleFuncsInDir(commandsDir, "commands"))
	maps.Copy(declared, s.handleFuncsInDir(queriesDir, "queries"))

	// Parse mediator.go and collect all RegisterHandler call arguments.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mediatorFile, nil, 0)
	s.Require().NoError(err, "parsing %s", mediatorFile)

	registered := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// The call may be RegisterHandler[T, U](...) which wraps the selector in
		// an IndexExpr or IndexListExpr. Unwrap to find the selector name.
		fun := callExpr.Fun
		switch idx := fun.(type) {
		case *ast.IndexExpr:
			fun = idx.X
		case *ast.IndexListExpr:
			fun = idx.X
		}
		sel, ok := fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "RegisterHandler" {
			return true
		}
		// Second argument is the handler function reference, e.g. commands.HandleFoo.
		if len(callExpr.Args) < 2 {
			return true
		}
		argSel, ok := callExpr.Args[1].(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkgIdent, ok := argSel.X.(*ast.Ident)
		if !ok {
			return true
		}
		registered[pkgIdent.Name+"."+argSel.Sel.Name] = true
		return true
	})

	for name := range declared {
		s.True(registered[name], "handler %q is declared but not registered with the mediator in setup/mediator.go", name)
	}
	for name := range registered {
		s.True(declared[name], "handler %q is registered in setup/mediator.go but not declared in commands/ or queries/", name)
	}
}
