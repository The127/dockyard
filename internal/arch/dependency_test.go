package arch_test

import (
	"go/parser"
	"go/token"
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
