package main

import (
	"bytes"
	"os"
	"text/template"

	"github.com/orange-cloudfoundry/promconsulfetcher/userdocs"
)

func main() {
	buf := &bytes.Buffer{}
	userdoc, _ := userdocs.Templates.ReadFile("templates/how-to-use.md")
	tpl, err := template.New("").Parse(string(userdoc))
	if err != nil {
		panic(err)
	}
	err = tpl.Execute(buf, struct {
		BaseURL string
	}{"my.promconsulfetcher.com"})
	os.Stdout.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
}
