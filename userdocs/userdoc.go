package userdocs

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*
var Templates embed.FS

//go:embed assets/*
var Assets embed.FS

type UserDoc struct {
	baseUrl string
}

var mainTpl *template.Template

func NewUserDoc(baseUrl string) *UserDoc {
	var err error
	mainFile, _ := Templates.ReadFile("templates/main.html")
	mainTpl, err = template.New("main.html").Funcs(tplfuncs).Parse(string(mainFile))
	if err != nil {
		panic(fmt.Sprintf("Cannot parse template 'templates/main.html': %s", err.Error()))
	}
	files, err := Templates.ReadDir("templates")
	if err != nil {
		panic(fmt.Sprintf("Cannot find templates : %s", err.Error()))
	}
	for _, f := range files {
		if f.Name() == "main.html" {
			continue
		}
		tplTxt, _ := Templates.ReadFile("templates/" + f.Name())
		_, err := mainTpl.New(f.Name()).Funcs(tplfuncs).Parse(string(tplTxt))
		if err != nil {
			panic(fmt.Sprintf("Cannot parse template '%s': %s", f.Name(), err.Error()))
		}
	}
	return &UserDoc{
		baseUrl: baseUrl,
	}
}

func (d UserDoc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := mainTpl.Execute(buf, struct {
		BaseURL string
	}{d.baseUrl})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(buf.Bytes())
}
