package arch_test

import (
	"fmt"
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

// handleHttpErrorAllowlist lists known violations of the "HandleHttpError must
// be followed by a return" rule that have not yet been fixed.
// Keys are "relpath:line" where relpath is relative to internal/.
var handleHttpErrorAllowlist = map[string]bool{
	"handlers/ocihandlers/blobs.go:46": true,
	"handlers/ocihandlers/blobs.go:55": true,
}

type violation struct {
	relPath  string
	line     int
	funcName string
}

type HandlerErrorArchTestSuite struct {
	suite.Suite
	internalDir string
}

func TestHandlerErrorArchTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HandlerErrorArchTestSuite))
}

func (s *HandlerErrorArchTestSuite) SetupSuite() {
	_, thisFile, _, ok := runtime.Caller(0)
	s.Require().True(ok, "runtime.Caller failed")
	s.internalDir = filepath.Join(filepath.Dir(thisFile), "..")
}

// goFilesUnderHandlers recursively collects all non-test .go files under
// internal/handlers/ and returns them as a map of relpath->absPath.
func (s *HandlerErrorArchTestSuite) goFilesUnderHandlers() map[string]string {
	handlersDir := filepath.Join(s.internalDir, "handlers")
	result := make(map[string]string)
	err := filepath.WalkDir(handlersDir, func(path string, d os.DirEntry, err error) error {
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

// isHandleHttpErrorCall reports whether expr is a call to HandleHttpError.
func isHandleHttpErrorCall(expr ast.Expr) bool {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "HandleHttpError"
}

// checkBlock walks all BlockStmt nodes within root and for each statement that
// is a call to HandleHttpError it asserts that the very next statement (if any)
// is a return. Violations are appended to the returned slice.
func checkBlock(root ast.Node, fset *token.FileSet, relPath string, funcName string) []violation {
	var violations []violation
	ast.Inspect(root, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		stmts := block.List
		for i, stmt := range stmts {
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				continue
			}
			if !isHandleHttpErrorCall(exprStmt.X) {
				continue
			}
			if i+1 < len(stmts) {
				if _, isReturn := stmts[i+1].(*ast.ReturnStmt); !isReturn {
					pos := fset.Position(stmt.Pos())
					violations = append(violations, violation{
						relPath:  relPath,
						line:     pos.Line,
						funcName: funcName,
					})
				}
			}
		}
		return true
	})
	return violations
}

// TestHandleHttpErrorAlwaysFollowedByReturn verifies that every call to
// HandleHttpError in internal/handlers/ is immediately followed by a return
// statement (or is the last statement in its block).
func (s *HandlerErrorArchTestSuite) TestHandleHttpErrorAlwaysFollowedByReturn() {
	files := s.goFilesUnderHandlers()

	fset := token.NewFileSet()
	for relPath, absPath := range files {
		f, err := parser.ParseFile(fset, absPath, nil, 0)
		s.Require().NoError(err, "parsing %s", absPath)

		for _, decl := range f.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil {
				continue
			}
			violations := checkBlock(funcDecl.Body, fset, relPath, funcDecl.Name.Name)
			for _, v := range violations {
				key := fmt.Sprintf("%s:%d", v.relPath, v.line)
				if handleHttpErrorAllowlist[key] {
					continue
				}
				s.Fail("HandleHttpError not followed by return",
					"%s:%d — %s has HandleHttpError not followed by return", v.relPath, v.line, v.funcName)
			}
		}
	}
}
