package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"text/template"
)

type paramBinding struct {
	ParamName string
	ParamType string
	TagValue  string
}

type fieldInfo struct {
	typ string
	tag string
}

type methodInfo struct {
	URL    string `json:"url"`
	Auth   bool   `json:"auth,omitempty"`
	Method string `json:"method,omitempty"`

	Recv  string
	Name  string
	Param string
}

type serveInfo struct {
	Recv    string
	Actions []methodInfo
}

var (
	fnMap = template.FuncMap{
		"lower": strings.ToLower,
		"enum": func(s string) string {
			return strings.Join(strings.Split(s, "|"), ", ")
		},
	}

	mainTpl      = template.Must(template.ParseFiles("./templates/main.tpl"))
	functionTpl  = template.Must(template.ParseFiles("./templates/function.tpl"))
	serveHTTPTpl = template.Must(template.New("serve").Parse(`
	func (srv *{{.Recv}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		{{range .Actions}}
		case "{{.URL}}": srv.handle{{.Name}}(w, r)
		{{end -}}
		default:
			body := response{Error: "unknown method"}
			http.Error(w, body.String(), http.StatusNotFound)
		}
	}
	`))

	requiredTpl = template.Must(template.New("required").Funcs(fnMap).Parse(`
	{{if eq .ParamType "int" -}}
	if t.{{.ParamName}} == 0 {
	{{else -}}
	if t.{{.ParamName}} == "" {
	{{end -}}
		return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("{{lower .ParamName}} must me not empty"),
		}
	}
	`))

	minTpl = template.Must(template.New("min").Funcs(fnMap).Parse(`
	{{if eq .ParamType "int" -}}
	if t.{{.ParamName}} < {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .ParamName}} must be >= {{.TagValue}}"),
		}
	}
	{{else -}}
	if len(t.{{.ParamName}}) < {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .ParamName}} len must be >= {{.TagValue}}"),
		}
	}
	{{end}}	 
	`))

	maxTpl = template.Must(template.New("max").Funcs(fnMap).Parse(`
	if t.{{.ParamName}} > {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .ParamName}} must be <= {{.TagValue}}"),
		}
	}	 
	`))

	enumTpl = template.Must(template.New("enum").Funcs(fnMap).Parse(`
	if !strings.Contains("|{{.TagValue}}|", "|"+t.{{.ParamName}}+"|") {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .ParamName}} must be one of [{{enum .TagValue}}]"),
		} 
	}
	`))

	defaultTpl = template.Must(template.New("default").Parse(`
	{{if eq .ParamType "int" -}}
	if t.{{.ParamName}} == 0 {
	{{else -}}
	if t.{{.ParamName}} == "" {
	{{end -}}
		t.{{.ParamName}} = "{{.TagValue}}"
	}
	`))

	bindTpl = template.Must(template.New("bind").Funcs(fnMap).Parse(`
	{{if eq .ParamType "int" -}}
	{{.ParamName}}, err := strconv.Atoi(q.Get("{{.TagValue}}"))
	if err != nil {
		return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("{{lower .ParamName}} must be int"),
		}
	}
	t.{{.ParamName}} = {{.ParamName}}
	{{else -}}
	t.{{.ParamName}} = q.Get("{{.TagValue}}")
	{{end}}`))

	ruleTemplates = []struct {
		rule string
		tpl  *template.Template
	}{
		{"default", defaultTpl},
		{"required", requiredTpl},
		{"min", minTpl},
		{"max", maxTpl},
		{"enum", enumTpl},
	}

	serveStructs = map[string]serveInfo{}
)

// код писать тут
func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])
	mainTpl.Execute(out, node.Name.Name)

	for _, decl := range node.Decls {
		switch t := decl.(type) {
		case *ast.FuncDecl:
			generateFunction(out, t)
		case *ast.GenDecl:
			for _, spec := range t.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %#T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %#T is not ast.StructType\n", currStruct)
					continue
				}

				processStruct(out, currType, currStruct)
			}
		}
	}

	for _, serveStruct := range serveStructs {
		serveHTTPTpl.Execute(out, serveStruct)
	}
}

func generateFunction(w io.Writer, fn *ast.FuncDecl) {
	if fn.Doc == nil || !strings.HasPrefix(fn.Doc.Text(), "apigen:api") {
		return
	}

	comment := strings.TrimPrefix(fn.Doc.Text(), "apigen:api")
	methodInfo := &methodInfo{}
	if err := json.Unmarshal([]byte(comment), methodInfo); err != nil {
		log.Panicln(err)
	}
	if methodInfo.Method == "" {
		methodInfo.Method = http.MethodGet + "|" + http.MethodPost
	}

	methodInfo.Name = fn.Name.Name
	if fn.Type.Params.List != nil {
		switch t := fn.Type.Params.List[1].Type.(type) {
		case *ast.Ident:
			methodInfo.Param = t.Name
		case *ast.SelectorExpr:
			methodInfo.Param = t.Sel.Name
		}
	}
	if fn.Recv != nil {
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			methodInfo.Recv = t.X.(*ast.Ident).Name
		case *ast.Ident:
			methodInfo.Recv = t.Name
		}
	}

	addAction(*methodInfo)
	functionTpl.Execute(w, methodInfo)
}

func processStruct(out io.Writer, t *ast.TypeSpec, s *ast.StructType) {
	needGenerate := false
	fields := make(map[string]fieldInfo)

	for _, field := range s.Fields.List {
		fieldName := field.Names[0].Name
		fieldType, ok := field.Type.(*ast.Ident)
		if !ok {
			fmt.Printf("SKIP %s.%s %T is not a primitive type", t.Name.Name, fieldName, field.Type)
		}

		if field.Tag != nil {
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			tagValue, ok := tag.Lookup("apivalidator")
			needGenerate = needGenerate || ok
			if ok {
				fields[fieldName] = fieldInfo{typ: fieldType.Name, tag: tagValue}
			}
		}
	}

	if needGenerate {
		generateBindMethod(out, t, fields)
		generateValidateMethod(out, t, fields)
	}
}

func generateBindMethod(out io.Writer, t *ast.TypeSpec, fields map[string]fieldInfo) {
	fmt.Fprintf(out, "func (t *%s) bind(q url.Values) error {", t.Name.Name)
	for key, field := range fields {
		rules := parseRules(field.tag)
		paramName, ok := rules["paramname"]
		if !ok || paramName == "" {
			paramName = key
		}
		if paramName == "-" {
			continue
		}

		bindTpl.Execute(out, &paramBinding{key, field.typ, strings.ToLower(paramName)})
	}
	fmt.Fprintln(out, "return nil")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func generateValidateMethod(out io.Writer, t *ast.TypeSpec, fields map[string]fieldInfo) {
	fmt.Fprintf(out, "func (t *%s) validate(q url.Values) error {", t.Name.Name)

	for key, field := range fields {
		rules := parseRules(field.tag)
		// iterate through templates to remain order: first assign default value, then apply other rules
		for _, ruleTemplate := range ruleTemplates {
			if param, ok := rules[ruleTemplate.rule]; ok {
				ruleTemplate.tpl.Execute(out, &paramBinding{key, field.typ, param})
			}
		}
	}

	fmt.Fprintln(out, "return nil")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)
}

func parseRules(tag string) map[string]string {
	rules := make(map[string]string)
	for _, rule := range strings.Split(tag, ",") {
		parts := strings.Split(rule, "=")
		if len(parts) == 1 {
			rules[parts[0]] = ""
		} else {
			rules[parts[0]] = parts[1]
		}
	}

	return rules
}

func addAction(info methodInfo) {
	serv, ok := serveStructs[info.Recv]
	if !ok {
		serveStructs[info.Recv] = serveInfo{Recv: info.Recv, Actions: make([]methodInfo, 0)}
	}

	serveStructs[info.Recv] = serveInfo{
		Recv:    info.Recv,
		Actions: append(serv.Actions, info),
	}
}
