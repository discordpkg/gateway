package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

func main() {
	file, err := parser.ParseFile(token.NewFileSet(), "internal/constants/events.go", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	var events []*eventInfo
	// Read the const key documentation from event/events.go
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

		name := valSpec.Names[0].Name
		doc := genDecl.Doc.Text()
		val := (valSpec.Values[0].(*ast.BasicLit)).Value
		if doc == "" {
			fmt.Fprintf(os.Stderr, "WARNING: events.%s has no docs! Please write some!\n", name)
		}

		events = append(events, &eventInfo{
			Name: name,
			Docs: doc,
			Val:  val,
		})
	}

	for _, event := range events {
		if event.Docs == "" {
			fmt.Fprintf(os.Stderr, "WARNING: %s is defined without documentation\n", event.Name)
		}
	}

	// Sort them alphabetically instead of the random iteration order from the maps.
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Name < events[j].Name
	})

	// And finally pass the event information to different templates to generate some files
	makeFile(events, "internal/generate/events/events.gohtml", "event/events_gen.go")
}

func makeFile(events []*eventInfo, tplFile, target string) {
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

	if err = ioutil.WriteFile(target, formatted, 0644); err != nil {
		panic(err)
	}
}

type eventInfo struct {
	Name string
	Docs string
	Val  string
}

func (e eventInfo) LowerCaseFirst() string {
	return string(unicode.ToLower(rune(e.Name[0]))) + string(e.Name[1:])
}

func (e eventInfo) String() string {
	return e.Name
}

func (e eventInfo) IsDiscordEvent() bool {
	return e.Docs != ""
}

func (e eventInfo) RenderDocs() string {
	if e.Docs == "" {
		return ""
	}

	str := strings.Replace(e.Docs, "\n", "\n// ", -1)
	return str[:len(str)-4]
}
