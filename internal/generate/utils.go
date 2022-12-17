package generate

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

func Generate(data any, templatePath, targetPath string) {
	templateName := path.Base(templatePath)
	tmpl, err := template.New(templateName).ParseFiles(templatePath)
	if err != nil {
		panic(fmt.Errorf("failed to parse template file: %w", err))
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		panic(fmt.Errorf("failed to generate code: %w", err))
	}

	//fmt.Println(string(b.Bytes()))

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		panic(fmt.Errorf("unable to format generated code: %w", err))
	}

	if err := os.WriteFile(targetPath, formatted, 0644); err != nil {
		panic(fmt.Errorf("failed to save generated code: %w", err))
	}
}

type MarkdownTableRow []string

type MarkdownTable struct {
	Title  string
	Header MarkdownTableRow
	Rows   []MarkdownTableRow
}

func (mdt *MarkdownTable) String() string {
	data := fmt.Sprintln(mdt.Title)
	data += fmt.Sprintln(mdt.Header)
	for row := range mdt.Rows {
		data += fmt.Sprintln(row)
	}

	return data
}

type MarkdownTableBuilder struct {
	title string
	lines []string
}

func (mdt *MarkdownTableBuilder) Transform() *MarkdownTable {
	cleanupRow := func(line string) MarkdownTableRow {
		var row MarkdownTableRow
		for _, cell := range strings.Split(line, "|") {
			cell = strings.Trim(cell, " ")
			if cell == "" || cell == "\n" {
				continue
			}

			row = append(row, cell)
		}

		return row
	}
	cleanupTitle := func(title string) string {
		title = strings.Trim(title, "#")
		title = strings.Trim(title, " ")
		title = strings.Trim(title, "\n")
		return title
	}

	if !strings.Contains(mdt.lines[1], "-") {
		panic("expected to row to be vertical table delimiter")
	}

	table := &MarkdownTable{
		Title:  cleanupTitle(mdt.title),
		Header: cleanupRow(mdt.lines[0]),
	}
	for _, line := range mdt.lines[2:] {
		table.Rows = append(table.Rows, cleanupRow(line))
	}
	return table
}

func (mdt *MarkdownTableBuilder) SetTitle(line string) {
	mdt.title = line
}

func (mdt *MarkdownTableBuilder) AddLine(line string) {
	mdt.lines = append(mdt.lines, line)
}

func copyLine(b []byte) []byte {
	buffer := make([]byte, len(b))
	copy(buffer, b)
	return buffer
}

type TableNameFilter func(name string) bool

func AnyTable(names []string) TableNameFilter {
	return func(name string) bool {
		for i := range names {
			if strings.ToLower(name) == names[i] {
				return true
			}
		}
		return false
	}
}

func ExtractMarkdownTables(markdownFilePath string, filter TableNameFilter) []*MarkdownTable {
	tableRegexp := regexp.MustCompile(`((\|[^|\r\n]*)+\|(\r?\n|\r)?)+`)
	titleRegexp := regexp.MustCompile(`([^\n]#+ [A-Za-z]+[^\n])`)

	f, err := os.Open(markdownFilePath)
	if err != nil {
		panic(fmt.Errorf("unable to open file: %w", err))
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	builder := &MarkdownTableBuilder{}
	var tables []*MarkdownTable
	for {
		line, err := rd.ReadBytes('\n')

		if titleRegexp.Match(line) && len(builder.lines) == 0 {
			builder.SetTitle(string(copyLine(line)))
		} else if tableRegexp.Match(line) {
			builder.AddLine(string(copyLine(line)))
		} else {
			if builder.title != "" && len(builder.lines) > 0 {
				var table *MarkdownTable
				table, builder = builder.Transform(), &MarkdownTableBuilder{}

				if filter != nil && !filter(table.Title) {
					continue
				}
				tables = append(tables, table)
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return tables
}
