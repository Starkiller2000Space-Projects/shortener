package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"iter"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/denormal/go-gitignore"
	"github.com/max-marek-projects/shortener/internal/logger"
	"go.uber.org/zap"
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

// iterPackagesRecursive iterates through directory and yields all dirs that are not ignored by gitignore
// Expects root directory and gitignore handler
// Returns iterator, which iterates over exch package yielding files names and package directory
func iterPackagesRecursive(root string, ignore gitignore.GitIgnore) iter.Seq2[[]string, string] {
	return func(yield func([]string, string) bool) {
		yieldPackageRecursive(root, root, ignore, yield)
	}
}

// yieldPackageRecursive handles directory during recursive iteration
// Expects root directory, current directory, gitignore handler and yield function
// Returns whether or not loop should be continued
func yieldPackageRecursive(root, dir string, ignore gitignore.GitIgnore, yield func([]string, string) bool) bool {
	logger.Log.Debug("iterating directory", zap.String("dir", dir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	var goFiles []string
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return true
		}
		match := ignore.Relative(rel, entry.IsDir())
		if match != nil {
			if match.Ignore() {
				logger.Log.Debug(
					"excluded path because of pattern",
					zap.String("path", rel), zap.Any("match", match), zap.Any("position", match.Position()),
				)
				continue
			}
		}
		if entry.IsDir() {
			if !yieldPackageRecursive(root, path, ignore, yield) {
				return false
			}
			continue
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") && filepath.Base(path) != resetFileName {
			goFiles = append(goFiles, path)
		}

	}
	if len(goFiles) > 0 {
		logger.Log.Debug("found do package", zap.String("dir", dir))
		if !yield(goFiles, dir) {
			return false
		}
	}
	return true
}

// findResetStructs searches for all structs marked with reset comment
// Expects filename to find marked structs in
// Returns package name, array of structs and error if any
func findResetStructs(filename string) (string, []*ast.TypeSpec, *types.Info, error) {
	logger.Log.Debug("finding resets in file", zap.String("dir", filename))
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return "", nil, nil, err
	}
	var result []*ast.TypeSpec
	packageName := file.Name.Name
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		if !hasGenerateReset(genDecl.Doc) {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)

			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				result = append(result, typeSpec)
			}
		}
	}

	// parse file into types
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{
		Importer: importer.Default(),
	}

	_, err = conf.Check(
		file.Name.Name,
		fileSet,
		[]*ast.File{file},
		info,
	)
	if err != nil {
		return "", nil, nil, err
	}

	return packageName, result, info, nil
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
func generateResetFile(packageName string, structs map[*types.Info][]*ast.TypeSpec) (*ast.File, error) {
	file := &ast.File{
		Name: ast.NewIdent(packageName),
	}

	for typesInfo, fileStructs := range structs {
		for _, singleStruct := range fileStructs {
			logger.Log.Debug("generating Reset method for struct", zap.String("name", singleStruct.Name.Name))
			fn, err := generateResetFunc(singleStruct, typesInfo)
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
func generateResetFunc(typeSpec *ast.TypeSpec, typesInfo *types.Info) (*ast.FuncDecl, error) {
	structType := typeSpec.Type.(*ast.StructType)
	receiverName := "r"
	var body []ast.Stmt
	for _, field := range structType.Fields.List {
		t := typesInfo.TypeOf(field.Type)
		for _, name := range field.Names {
			currentResetStatement, err := resetStatement(receiverName, name.Name, t, false)
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

// zeroValue returns zero value ast node for various field types
// Expects receiver name, field name, field type from go/types, whether variable is a pointer or not
func resetStatement(receiver, field string, fieldType types.Type, pointer bool) (ast.Stmt, error) {
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
		return resetStatement(receiver, field, fieldType.Underlying(), pointer)

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
		underlyingStatement, err := resetStatement(receiver, field, fieldType.Elem(), true)
		if err != nil {
			return nil, err
		}
		if underlyingStatement != nil {
			ifStatement.Body.List = append(ifStatement.Body.List, underlyingStatement)
			return ifStatement, nil
		}
		return nil, nil
	case *types.Struct:
		return &ast.IfStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("resetter"), ast.NewIdent("ok")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.TypeAssertExpr{
						X: &ast.CallExpr{
							Fun: ast.NewIdent("any"),
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   ast.NewIdent(receiver),
									Sel: ast.NewIdent(field),
								},
							},
						},
						Type: &ast.InterfaceType{
							Methods: &ast.FieldList{
								List: []*ast.Field{
									{
										Names: []*ast.Ident{ast.NewIdent("Reset")},
										Type: &ast.FuncType{
											Params: &ast.FieldList{},
										},
									},
								},
							},
						},
					},
				},
			},
			Cond: ast.NewIdent("ok"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("resetter"),
								Sel: ast.NewIdent("Reset"),
							},
						},
					},
				},
			},
		}, nil
	case *types.Interface:
		if pointer {
			return nil, fmt.Errorf("reset not implemented for interface pointers: %v", fieldType)
		}
		return &ast.IfStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("resetter"), ast.NewIdent("ok")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.TypeAssertExpr{
						X: &ast.SelectorExpr{
							X:   ast.NewIdent(receiver),
							Sel: ast.NewIdent(field),
						},
						Type: &ast.InterfaceType{
							Methods: &ast.FieldList{
								List: []*ast.Field{
									{
										Names: []*ast.Ident{ast.NewIdent("Reset")},
										Type: &ast.FuncType{
											Params: &ast.FieldList{},
										},
									},
								},
							},
						},
					},
				},
			},
			Cond: ast.NewIdent("ok"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("resetter"),
								Sel: ast.NewIdent("Reset"),
							},
						},
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown type: %v", fieldType)
	}
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
	ignore, err := gitignore.NewRepository(rootDir)
	if err != nil {
		return err
	}
	for files, packagePath := range iterPackagesRecursive(rootDir, ignore) {
		var resetStructs = make(map[*types.Info][]*ast.TypeSpec)
		var packageName string
		for _, file := range files {
			filePackageName, fileResetStructs, typesInfo, err := findResetStructs(file)
			if err != nil {
				return err
			}
			if packageName == "" {
				packageName = filePackageName
			} else if packageName != filePackageName {
				return fmt.Errorf("package names do not match: %s, %s", packageName, filePackageName)
			}
			if len(fileResetStructs) == 0 {
				continue
			}
			resetStructs[typesInfo] = fileResetStructs
		}
		if len(resetStructs) == 0 {
			continue
		}
		fileData, err := generateResetFile(packageName, resetStructs)
		if err != nil {
			return err
		}
		err = writeGenerated(filepath.Join(packagePath, resetFileName), fileData)
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
	err := logger.Initialize("INFO")
	if err != nil {
		log.Fatalf("unable to initialize logger: %v", err)
	}
	err = makeResets()
	if err != nil {
		log.Fatalf("Error creating resets: %v", err)
	}
}
