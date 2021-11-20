package generate

import (
	"bytes"
	"go/format"
	"path"
	"strings"
	"text/template"
)

func FmtRemovePointerStrict(s string) string {
	if s == "" {
		panic("can't dereference an empty variable name")
	}
	if s[0] != '*' {
		panic("not a pointer")
	}

	return s[1:]
}

func FmtRemovePointer(s string) string {
	if s != "" && s[0] == '*' {
		return s[1:]
	}
	return s
}

func FmtDecapitalize(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

func GoCode(data interface{}, templateFile, targetFile string) (formattedCode []byte, err error) {
	templateConfiguration := template.Must(template.
		New(path.Base(templateFile)).
		Funcs(template.FuncMap{
			"ToUpper":             strings.ToUpper,
			"ToLower":             strings.ToLower,
			"Decapitalize":        FmtDecapitalize,
			"RemovePointer":       FmtRemovePointer,
			"RemovePointerStrict": FmtRemovePointerStrict,
		}).
		ParseFiles(templateFile))

	// Execute the template, inserting all the event information
	var b bytes.Buffer
	if err = templateConfiguration.Execute(&b, data); err != nil {
		return nil, err
	}

	if formattedCode, err = format.Source(b.Bytes()); err != nil {
		return nil, err
	}

	return formattedCode, nil
}
