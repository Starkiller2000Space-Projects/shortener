package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasGenerateReset(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected bool
	}{
		{
			name:     "with comment",
			comment:  "// generate:reset",
			expected: true,
		},
		{
			name:     "with comment and additional",
			comment:  "// generate:reset some extra",
			expected: true,
		},
		{
			name:     "without comment",
			comment:  "// some comment",
			expected: false,
		},
		{
			name:     "empty comment",
			comment:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cg *ast.CommentGroup
			if tt.comment != "" {
				cg = &ast.CommentGroup{
					List: []*ast.Comment{
						{Text: tt.comment},
					},
				}
			}
			result := hasGenerateReset(cg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateResetFunc(t *testing.T) {
	src := `
package testpkg


type SomeInterface interface {
	Reset()
}

type OtherStruct struct {
}

func (r *OtherStruct) Reset() {}


type ResetableStruct struct {
    i     int
	iP    *int
	b     bool
	bP    *bool
	f     float64
	fP    *float64
	c     complex64
	cP    *complex64
    str   string
    strP  *string
    s     []int
	sP    *[]int
    m     map[string]string
	mP    *map[string]string
    child *ResetableStruct
	ifVar SomeInterface
	ot    OtherStruct
	otP   *OtherStruct
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, err)

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	_, err = conf.Check("testpkg", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	var typeSpec *ast.TypeSpec
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts := spec.(*ast.TypeSpec)
			if ts.Name.Name == "ResetableStruct" {
				typeSpec = ts
				break
			}
		}
	}
	require.NotNil(t, typeSpec)
	fn, err := generateResetFunc(typeSpec, info, map[types.Object]bool{info.Defs[typeSpec.Name]: true}, map[string]bool{}, nil)
	require.NoError(t, err)

	assert.Equal(t, "Reset", fn.Name.Name)

	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), fn)
	require.NoError(t, err)
	code := buf.String()

	assert.Contains(t, code, `
	if r == nil {
		return
	}`)

	assert.Contains(t, code, "r.i = 0")
	assert.Contains(t, code, "r.b = false")
	assert.Contains(t, code, "r.f = 0.0")
	assert.Contains(t, code, "r.c = 0i")
	assert.Contains(t, code, `r.str = ""`)

	assert.Contains(t, code, `
	if r.iP != nil {
		*r.iP = 0
	}`)
	assert.Contains(t, code, `
	if r.bP != nil {
		*r.bP = false
	}`)
	assert.Contains(t, code, `
	if r.fP != nil {
		*r.fP = 0.0
	}`)
	assert.Contains(t, code, `
	if r.cP != nil {
		*r.cP = 0i
	}`)
	assert.Contains(t, code, `
	if r.strP != nil {
		*r.strP = ""
	}`)

	assert.Contains(t, code, "r.s = r.s[:0]")
	assert.Contains(t, code, `
	if r.sP != nil {
		*r.sP = (*r.sP)[:0]
	}`)

	assert.Contains(t, code, "clear(r.m)")
	assert.Contains(t, code, `
	if r.mP != nil {
		clear(*r.mP)
	}`)

	assert.Contains(t, code, `
	if r.child != nil {
		r.child.Reset()
	}`)

	assert.Contains(t, code, `
	if r.ifVar != nil {
		r.ifVar.Reset()
	}`)

	assert.Contains(t, code, "r.ot.Reset()")
	assert.Contains(t, code, `
	if r.otP != nil {
		r.otP.Reset()
	}`)
}
