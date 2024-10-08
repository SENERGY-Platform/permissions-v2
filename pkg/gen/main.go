/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"slices"
	"strings"
	"text/template"
)

//go:generate go run main.go
//go:generate go install github.com/swaggo/swag/cmd/swag@latest
//go:generate swag init --instanceName permissionsv2 -o ../../docs --parseDependency -d .. -g ../pkg/api/api.go

func main() {
	GenerateClientDocFile()
}

func GenerateClientDocFile() {
	log.Println("Generate pkg/client/swaggo_comments_file.tmpl")
	f, err := parser.ParseDir(token.NewFileSet(), "../api", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	list := []Element{}

	for _, method := range FilterMethods(f) {
		name := method.Name.String()
		doc := method.Doc.Text()

		if strings.Contains(doc, "health check") {
			continue //ignore health check
		}

		//change route comment
		splitByRouter := strings.SplitN(doc, "@Router", 2)
		if len(splitByRouter) >= 2 {
			parts := strings.Split(splitByRouter[1], "/")
			if len(parts) >= 2 && strings.TrimSpace(parts[0]) == "" {
				parts = slices.Insert(parts, 1, "{{.PrefixWithoutSlash}}")
			} else {
				parts = slices.Insert(parts, 0, "/{{.PrefixWithoutSlash}}")
			}
			splitByRouter[1] = strings.Join(parts, "/")
		}
		changedDoc := strings.Join(splitByRouter, "@Router")
		lines := strings.Split(changedDoc, "\n")
		comments := []string{}
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				comments = append(comments, "// "+strings.ReplaceAll(line, name, "Generated"+name))
			}
		}
		list = append(list, Element{
			Name: name,
			Doc:  strings.Join(comments, "\n"),
		})
	}

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	location := "../client/swaggo_comments_file.tmpl"

	file, err := os.OpenFile(location, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("unable to close output file %v %w", location, err))
		}
	}()

	err = t.Execute(file, map[string]interface{}{"Data": list, "PackageName": "{{.PackageName}}"})
	if err != nil {
		panic(err)
	}
}

const tmpl = `
//GENERATED FILE -- DO NOT EDIT
//base template generated in github.com/SENERGY-Platform/permissions-v2 root dir, with 'go generate ./...'
//result file may be generated by client.GenerateGoFileWithSwaggoCommentsForPermissionsRequestsForwarding()

/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package {{.PackageName}}


{{range $element := .Data}}
{{$element.Doc}}
func Generated{{$element.Name}}(){
	//this method is only used as anchor for swagger documentation
	panic("this method is only used as anchor for swagger documentation")
}


{{end}}
`

type Element struct {
	Name string
	Doc  string
}

func FilterMethods(dirAst map[string]*ast.Package) (result []*ast.FuncDecl) {
	for _, packageAst := range dirAst {
		for _, fileAst := range packageAst.Files {
			for _, decl := range fileAst.Decls {
				fdecl, ok := decl.(*ast.FuncDecl)
				if ok && fdecl.Recv != nil && len(fdecl.Recv.List) > 0 {
					result = append(result, fdecl)
				}
			}
		}
	}
	return result
}
