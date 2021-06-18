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
	"sort"
	"strings"
	"text/template"
)

func main() {
	//file, err := parser.ParseFile(token.NewFileSet(), "/home/anders/dev/discordgateway/internal/constants/intents.go", nil, parser.ParseComments)
	file, err := parser.ParseFile(token.NewFileSet(), "internal/constants/intents.go", nil, 0)
	if err != nil {
		panic(err)
	}

	var intents []*intentInfo
	for name, item := range file.Scope.Objects {
		if item.Kind != ast.Var {
			continue
		}

		val := item.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
		data := make(map[string]interface{})
		for i := 0; i < len(val.Elts); i++ {
			pair := val.Elts[i].(*ast.KeyValueExpr)
			key := fmt.Sprint(pair.Key)
			data[key] = pair.Value
		}

		events := unwrapEvents(data["Events"])
		//intent := data["Intent"].(*ast.Ident).Obj.Data.(int)

		intents = append(intents, &intentInfo{
			Name:   name,
			Intent: "constants." + name + "Val",
			Events: fmt.Sprintf("[]event.Type{%s}", strings.Join(events, ",")),
		})
	}

	// Sort them alphabetically instead of the random iteration order from the maps.
	sort.SliceStable(intents, func(i, j int) bool {
		return intents[i].Name < intents[j].Name
	})

	makeFile(intents, "internal/generate/intents/intents.gohtml", "intent/intents_gen.go")
}

func decodeSelectorExpr(sel *ast.SelectorExpr) string {
	pkg := sel.X.(*ast.Ident).Name
	val := sel.Sel.Name
	return pkg + "." + val
}

func unwrapEvents(evts interface{}) (names []string) {
	if sel, ok := evts.(*ast.SelectorExpr); ok {
		names = append(names, decodeSelectorExpr(sel))
		return names
	}

	comps := evts.(*ast.CompositeLit)
	for i := range comps.Elts {
		elt := comps.Elts[i].(*ast.SelectorExpr)
		names = append(names, elt.X.(*ast.Ident).Name+"."+elt.Sel.Name)
	}

	return names
}

func makeFile(events []*intentInfo, tplFile, target string) {
	// Open & parse our template
	tpl := template.Must(template.New(path.Base(tplFile)).ParseFiles(tplFile))

	// Execute the template, inserting all the event information
	var b bytes.Buffer
	if err := tpl.Execute(&b, events); err != nil {
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

type intentInfo struct {
	Name   string
	Intent string
	Events string
}

func (e intentInfo) String() string {
	return e.Name
}

func (e intentInfo) IsDM() bool {
	return strings.HasPrefix(e.Name, "DirectMessage")
}
