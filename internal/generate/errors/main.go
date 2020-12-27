package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
)

func main() {
	file, err := parser.ParseFile(token.NewFileSet(), "error.go", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	errorTypes := map[string]*errorType{}
	for name, item := range file.Scope.Objects {
		if item.Kind != ast.Typ {
			continue
		}

		if e, ok := item.Decl.(*ast.TypeSpec).Type.(*ast.Ident); ok {
			if e.Name != "DiscordErrorCode" {
				continue
			}

			errorTypes[name] = &errorType{
				Name: name,
			}
		}
	}

	// Read error const and create a string representation using the comment
	for _, item := range file.Decls {
		// Check if this is a GenDecl and if it has at least 1 spec
		genDecl, ok := item.(*ast.GenDecl)
		if !ok || len(genDecl.Specs) == 0 {
			continue
		}
		// Check if it is a ValueSpec and check if it has at least 1 name
		valSpec, ok := genDecl.Specs[0].(*ast.ValueSpec)
		if !ok || len(valSpec.Names) == 0 {
			continue
		}

		// we need the type for the consts here
		// check the first value spec and fetch it
		//
		// after we check if we have a register for it
		if t, ok := valSpec.Type.(*ast.Ident); !ok {
			continue
		} else if _, ok := errorTypes[t.Name]; !ok {
			continue
		}

		register := valSpec.Type.(*ast.Ident).Name
		errorT := errorTypes[register]

		// fetch all the error codes for this type
		for i, spec := range genDecl.Specs {
			vspec := spec.(*ast.ValueSpec)
			if i > 0 && vspec.Type != nil {
				panic("expected value type to be unknown - adapt code")
			}

			name := vspec.Names[0].Name
			if name == "_" {
				continue
			}
			comment := vspec.Doc.Text()
			if comment == "" {
				panic(fmt.Errorf("ERROR: %s has no docs! Please write some!\n", name))
			}
			description := strings.TrimPrefix(comment, "// "+name+" ")
			description = strings.TrimSuffix(description, "\n")

			errorT.Codes = append(errorT.Codes, &errorCode{
				Name:        name,
				Description: description,
			})
		}
	}

	// now we need to find the struct that wraps the error code
	for name, item := range file.Scope.Objects {
		if item.Kind != ast.Typ {
			continue
		}

		structInfo, ok := item.Decl.(*ast.TypeSpec).Type.(*ast.StructType)
		if !ok {
			continue
		}

		fields := structInfo.Fields.List
		var code *ast.Field
		for _, field := range fields {
			if field.Names[0].Name != "Code" {
				continue
			}

			code = field
			break
		}
		if code == nil {
			continue
		}

		typename := code.Type.(*ast.Ident).Name
		if register, ok := errorTypes[typename]; ok {
			register.Struct = name
		}
	}

	// And finally pass the event information to different templates to generate some files
	var types []*errorType
	for _, v := range errorTypes {
		types = append(types, v)
	}
	makeFile(types, "internal/generate/errors/errors.gohtml", "error_gen.go")
}

func makeFile(errors []*errorType, tplFile, target string) {
	// Open & parse our template
	tpl := template.Must(template.New(path.Base(tplFile)).ParseFiles(tplFile))

	// Execute the template, inserting all the event information
	var b bytes.Buffer
	if err := tpl.Execute(&b, errors); err != nil {
		panic(err)
	}

	// Format it according to gofmt standards
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}

	// And write it.
	if err = ioutil.WriteFile(target, formatted, 0644); err != nil {
		panic(err)
	}
}

type errorType struct {
	Name   string
	Codes  []*errorCode
	Struct string
}

func (e errorType) String() string {
	return e.Name
}

type errorCode struct {
	Name        string
	Description string
}
