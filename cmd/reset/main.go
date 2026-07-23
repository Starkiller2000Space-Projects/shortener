package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/max-marek-projects/shortener/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"
)

var resetFileName string = "reset.gen.go"

// findRootDir searches for go mod file in parent directories of current directory
// Returns root directory and error if any
func findRootDir(dir string) (string, error) {
	for {
		logger.Log.Debug("analyzing directory", zap.String("dir", dir))
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			logger.Log.Info("found root directory", zap.String("dir", dir))
			return dir, nil
		}
		if dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("go.mod file not found")
}

// findResetStructs searches for all structs marked with reset comment
// Expects filename to find marked structs in
// Returns package name, array of structs and error if any
func findResetStructs(file *ast.File, typesInfo *types.Info) ([]*ResetStruct, error) {
	var result []*ResetStruct
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		hasResetInBlock := hasGenerateReset(genDecl.Doc)
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			if !hasResetInBlock && !hasGenerateReset(typeSpec.Doc) {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				structInfo, ok := typesInfo.Defs[typeSpec.Name]
				if !ok {
					return nil, fmt.Errorf("struct %s not found", typeSpec.Name)
				}
				result = append(result, &ResetStruct{Type: typeSpec, Obj: structInfo})
			}
		}
	}
	return result, nil
}

// hasGenerateReset checks if struct declaration has "reset" comment
// Expects comment over struct definition
// Returns true/false whether or not comment has desired marker
func hasGenerateReset(commentGroup *ast.CommentGroup) bool {
	if commentGroup == nil {
		return false
	}
	for _, commentLine := range commentGroup.List {
		if strings.HasPrefix(commentLine.Text, "// generate:reset") {
			return true
		}
	}
	return false
}

// generateResetFile generates reset file for current package
// Expects go package name, array of found nodes for structs definitions
// Returns reset file definition and error if any
func generateResetFile(packageName string, structs map[*types.Info][]*ast.TypeSpec, allResetTypes map[types.Object]bool) (*ast.File, error) {
	file := &ast.File{
		Name: ast.NewIdent(packageName),
	}

	for typesInfo, fileStructs := range structs {
		for _, singleStruct := range fileStructs {
			logger.Log.Debug("generating Reset method for struct", zap.String("name", singleStruct.Name.Name))
			fn, err := generateResetFunc(singleStruct, typesInfo, allResetTypes)
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, fn)
			logger.Log.Info("generated Reset method for struct", zap.String("name", singleStruct.Name.Name))
		}
	}

	return file, nil
}

// generateResetFunc generates reset func for found struct
// Expects struct definition
// Returns Reset function declaration node
func generateResetFunc(typeSpec *ast.TypeSpec, typesInfo *types.Info, allResetTypes map[types.Object]bool) (*ast.FuncDecl, error) {
	structType := typeSpec.Type.(*ast.StructType)
	receiverName := "r"
	var body []ast.Stmt
	for _, field := range structType.Fields.List {
		t := typesInfo.TypeOf(field.Type)
		for _, name := range field.Names {
			currentResetStatement, err := resetStatement(receiverName, name.Name, t, false, false, allResetTypes)
			if err != nil {
				return nil, err
			}
			if currentResetStatement != nil {
				body = append(
					body,
					currentResetStatement,
				)
			}
		}
	}
	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						ast.NewIdent(receiverName),
					},
					Type: &ast.StarExpr{
						X: ast.NewIdent(typeSpec.Name.Name),
					},
				},
			},
		},

		Name: ast.NewIdent("Reset"),

		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: &ast.BlockStmt{List: append([]ast.Stmt{nilCheck(receiverName)}, body...)},
	}, nil
}

// nilCheck creates ast Node that checks whether pointer points to nil
// Expects receiver argument name
// Returns ast node
func nilCheck(receiverArgName string) ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent(receiverArgName),
			Op: token.EQL,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{},
			},
		},
	}
}

// resetStatement create reset statement for single field
// Expects receiver argument name, field name and ast node with field type definition
// Returns ast node for resetting field
func assignmentStatement(receiver, field string, pointer bool, zero ast.Expr) ast.Stmt {
	var expr ast.Expr = &ast.SelectorExpr{
		X:   ast.NewIdent(receiver),
		Sel: ast.NewIdent(field),
	}
	if pointer {
		expr = &ast.StarExpr{X: expr}
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{expr},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			zero,
		},
	}
}

func hasResetMethod(typ types.Type) bool {
	mset := types.NewMethodSet(typ)
	for i := 0; i < mset.Len(); i++ {
		meth := mset.At(i).Obj().(*types.Func)
		if meth.Name() == "Reset" {
			sig := meth.Type().(*types.Signature)
			if sig.Params().Len() == 0 {
				return true
			}
		}
	}
	ptr := types.NewPointer(typ)
	mset = types.NewMethodSet(ptr)
	for i := 0; i < mset.Len(); i++ {
		meth := mset.At(i).Obj().(*types.Func)
		if meth.Name() == "Reset" {
			sig := meth.Type().(*types.Signature)
			if sig.Params().Len() == 0 {
				return true
			}
		}
	}
	return false
}

// zeroValue returns zero value ast node for various field types
// Expects receiver name, field name, field type from go/types, whether variable is a pointer or not
func resetStatement(receiver, field string, fieldType types.Type, pointer bool, hasReset bool, allResetTypes map[types.Object]bool) (ast.Stmt, error) {
	logger.Log.Debug("analyzing type", zap.Any("type", fieldType))
	switch fieldType := fieldType.(type) {
	case *types.Basic:
		switch fieldType.Name() {
		case "string":
			return assignmentStatement(receiver, field, pointer, &ast.BasicLit{
				Kind:  token.STRING,
				Value: `""`,
			}), nil
		case "bool":
			return assignmentStatement(receiver, field, pointer, ast.NewIdent("false")), nil
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			return assignmentStatement(receiver, field, pointer, &ast.BasicLit{Kind: token.INT, Value: "0"}), nil
		case "float32", "float64":
			return assignmentStatement(receiver, field, pointer, &ast.BasicLit{Kind: token.FLOAT, Value: "0.0"}), nil
		case "complex64", "complex128":
			return assignmentStatement(receiver, field, pointer, &ast.BasicLit{Kind: token.IMAG, Value: "0i"}), nil
		default:
			return nil, fmt.Errorf("unknown basic type: %s", fieldType.Name())
		}

	case *types.Slice:
		var expr ast.Expr = &ast.SelectorExpr{
			X:   ast.NewIdent(receiver),
			Sel: ast.NewIdent(field),
		}
		if pointer {
			expr = &ast.StarExpr{X: expr}
		}
		return assignmentStatement(receiver, field, pointer, &ast.SliceExpr{
			X:    expr,
			Low:  nil,
			High: &ast.BasicLit{Kind: token.INT, Value: "0"},
		}), nil

	case *types.Map:
		var expr ast.Expr = &ast.SelectorExpr{
			X:   ast.NewIdent(receiver),
			Sel: ast.NewIdent(field),
		}
		if pointer {
			expr = &ast.StarExpr{X: expr}
		}
		return &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  ast.NewIdent("clear"),
				Args: []ast.Expr{expr},
			},
		}, nil

	case *types.Named:
		if _, ok := allResetTypes[fieldType.Obj()]; ok || hasResetMethod(fieldType) {
			return resetStatement(receiver, field, fieldType.Underlying(), pointer, true, allResetTypes)
		}
		return resetStatement(receiver, field, fieldType.Underlying(), pointer, hasReset, allResetTypes)
	case *types.Pointer:
		expr := &ast.SelectorExpr{
			X:   ast.NewIdent(receiver),
			Sel: ast.NewIdent(field),
		}
		ifStatement := &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  expr,
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{},
			},
		}
		underlyingStatement, err := resetStatement(receiver, field, fieldType.Elem(), true, hasReset, allResetTypes)
		if err != nil {
			return nil, err
		}
		if underlyingStatement != nil {
			ifStatement.Body.List = append(ifStatement.Body.List, underlyingStatement)
			return ifStatement, nil
		}
		return nil, nil
	case *types.Struct:
		expr := &ast.SelectorExpr{
			X:   ast.NewIdent(receiver),
			Sel: ast.NewIdent(field),
		}
		if hasReset || hasResetMethod(fieldType) {
			return &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   expr,
						Sel: ast.NewIdent("Reset"),
					},
				},
			}, nil
		}
		if pointer {
			return assignmentStatement(receiver, field, false, ast.NewIdent("nil")), nil
		}
		return nil, nil
	case *types.Interface:
		expr := &ast.SelectorExpr{
			X:   ast.NewIdent(receiver),
			Sel: ast.NewIdent(field),
		}
		if pointer {
			return nil, fmt.Errorf("reset not implemented for interface pointers: %v", fieldType)
		}
		if hasReset || hasResetMethod(fieldType) {
			return &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  expr,
					Op: token.NEQ,
					Y:  ast.NewIdent("nil"),
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   expr,
								Sel: ast.NewIdent("Reset"),
							},
						},
					}},
				},
			}, nil
		}
		return assignmentStatement(receiver, field, false, ast.NewIdent("nil")), nil
	default:
		return nil, fmt.Errorf("unknown type: %v", fieldType)
	}
}

type ResetStruct struct {
	Obj  types.Object
	Type *ast.TypeSpec
}

// makeResets parses files and generates reset functions for them
func makeResets() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	rootDir, err := findRootDir(currentDir)
	if err != nil {
		return err
	}
	cfg := &packages.Config{
		Mode: packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedCompiledGoFiles |
			packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, fmt.Sprintf("%s/...", rootDir))
	if err != nil {
		return err
	}
	var resetTypes = make(map[types.Object]bool)
	var packageResetStructs = make(map[*packages.Package]map[*types.Info][]*ast.TypeSpec)
	for _, pkg := range pkgs {
		var resetStructs = make(map[*types.Info][]*ast.TypeSpec)
		for _, file := range pkg.Syntax {
			fileResetStructs, err := findResetStructs(file, pkg.TypesInfo)
			if err != nil {
				return err
			}
			if len(fileResetStructs) == 0 {
				continue
			}
			for _, fileResetStruct := range fileResetStructs {
				resetStructs[pkg.TypesInfo] = append(resetStructs[pkg.TypesInfo], fileResetStruct.Type)
				resetTypes[fileResetStruct.Obj] = true
			}
		}
		if len(resetStructs) == 0 {
			continue
		}
		packageResetStructs[pkg] = resetStructs
	}
	for pkg, resetStructs := range packageResetStructs {
		fileData, err := generateResetFile(pkg.Name, resetStructs, resetTypes)
		if err != nil {
			return err
		}
		err = writeGenerated(filepath.Join(pkg.Dir, resetFileName), fileData)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeGenerated writes generated ast Nodes to desired location
func writeGenerated(filename string, f *ast.File) error {
	var buf bytes.Buffer
	err := format.Node(
		&buf,
		token.NewFileSet(),
		f,
	)
	if err != nil {
		return err
	}
	return os.WriteFile(
		filename,
		buf.Bytes(),
		0600,
	)
}

// entry point
// Initializes logger and creates files with reset functions
func main() {
	err := logger.Initialize("DEBUG")
	if err != nil {
		log.Fatalf("unable to initialize logger: %v", err)
	}
	err = makeResets()
	if err != nil {
		log.Fatalf("Error creating resets: %v", err)
	}
}
