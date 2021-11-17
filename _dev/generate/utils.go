package generate

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
)

func MakeFile(data interface{}, templateFile, targetFile string) (err error) {
	functions := template.FuncMap{
		"ToUpper":      strings.ToUpper,
		"ToLower":      strings.ToLower,
		"Decapitalize": func(s string) string { return strings.ToLower(s[0:1]) + s[1:] },
		"RemovePointer": func(s string) string {
			if s != "" && s[0] == '*' {
				return s[1:]
			}
			return s
		},
	}
	tmpl := template.Must(template.New(path.Base(templateFile)).Funcs(functions).ParseFiles(templateFile))

	// Execute the template, inserting all the event information
	var b bytes.Buffer
	if err = tmpl.Execute(&b, data); err != nil {
		return err
	}

	var formattedCode []byte
	if formattedCode, err = format.Source(b.Bytes()); err != nil {
		return err
	}

	if err = ioutil.WriteFile(targetFile, formattedCode, 0644); err != nil {
		return err
	}

	return nil
}
