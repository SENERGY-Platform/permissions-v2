/*
 * Copyright 2025 InfAI (CC SES)
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

package client

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

// GenerateGoFileWithSwaggoCommentsForEmbeddedPermissionsClient
// generates a go file with swaggo comments needed to generate swagger documentation
// - location: defines where the new go file should be generated (containing the file-name e.g. "../generated_permissions.go")
// - packageName: sets the package name of the generated file
// - prefix: is the prefix before the permission endpoints. may have "/" prefix und suffix (e.g. "/permissions/"). default = "permissions"
// - topicList: replaces endpoints with {topic} placeholder with explicit endpoints; may be nil to keep placeholder
// - pathFilter:
//   - function to filter undesired endpoints (like DELETE endpoints)
//   - may be nil to allow all endpoints
//   - pathFilter should return true to allow the endpoint
//   - no guarantee if the input path contains placeholders like {topic} and {id}, or if the placeholder is replaced
//   - the input path contains the prefix parameter to ensure compatibility with EmbedPermissionsClientIntoRouter
func GenerateGoFileWithSwaggoCommentsForEmbeddedPermissionsClient(packageName string, prefix string, location string, topicList []string, pathFilter func(method string, path string) bool) error {
	log.Println("Generate " + location)

	p, err := build.Import("github.com/SENERGY-Platform/permissions-v2/pkg/api", ".", build.ImportComment)
	if err != nil {
		return err
	}

	f, err := parser.ParseDir(token.NewFileSet(), p.Dir, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	prefix = strings.Trim(prefix, "/")

	list := getElements(f, prefix, topicList, pathFilter)

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(location, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("unable to close output file %v %w", location, err))
		}
	}()

	err = t.Execute(file, map[string]interface{}{"PackageName": packageName, "Data": list})
	if err != nil {
		return err
	}
	return nil
}

type element struct {
	Name string
	Doc  string
}

func getElements(f map[string]*ast.Package, prefixWithoutSlash string, topicList []string, filter func(method string, path string) bool) []element {
	list := []element{}
	for _, method := range filterMethods(f) {
		name := method.Name.String()
		doc := method.Doc.Text()

		if strings.Contains(name, "HealthCheck") {
			continue //ignore health check
		}

		var routeMethod, routePath string
		doc, routeMethod, routePath = updateEndpoint(doc, prefixWithoutSlash)
		if !filter(routeMethod, routePath) {
			continue
		}

		docsWithReplacedTopics := []element{}
		if topicList != nil {
			docsWithReplacedTopics = handleTopicList(doc, name, topicList)
		} else {
			docsWithReplacedTopics = append(list, element{
				Name: "Generated" + name,
				Doc:  doc,
			})
		}

		for _, sub := range docsWithReplacedTopics {
			sub.Doc = replaceMethodNameInDoc(sub.Doc, name, sub.Name)
			sub.Doc = prefixTags(sub.Doc, prefixWithoutSlash)
			list = append(list, sub)
		}

	}

	return list
}

func prefixTags(doc string, prefix string) string {
	lines := []string{}
	for line := range strings.Lines(doc) {
		if regexp.MustCompile(`\s*@Tags`).MatchString(line) {
			tags := []string{}
			for _, tag := range strings.Split(strings.TrimSpace(strings.Split(line, "@Tags")[1]), ",") {
				tags = append(tags, prefix+"-"+strings.TrimSpace(tag))
			}
			lines = append(lines, "// @Tags         "+strings.Join(tags, ","))
		} else {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return strings.Join(lines, "\n")
}

func handleTopicList(doc string, name string, list []string) (result []element) {
	if !strings.Contains(doc, "{topic}") {
		return []element{
			{
				Name: "Generated" + name,
				Doc:  doc,
			},
		}
	}
	lines := []string{}
	for line := range strings.Lines(doc) {
		switch {
		case regexp.MustCompile(`\s*@Param\s*topic\s*path`).MatchString(line):
			continue
		case regexp.MustCompile(`\s*@Summary`).MatchString(line):
			lines = append(lines, strings.ReplaceAll(line, "topic", "topic ({topic})"))
		case regexp.MustCompile(`\s*@Description`).MatchString(line):
			lines = append(lines, strings.ReplaceAll(line, "topic", "topic ({topic})"))
		case regexp.MustCompile(`\s*@Tags`).MatchString(line):
			lines = append(lines, `@Tags         {topic}`)
		default:
			lines = append(lines, line)
		}
	}
	doc = strings.Join(lines, "\n")
	for _, topic := range list {
		newDock := strings.ReplaceAll(doc, "{topic}", topic)
		suffix := topic
		suffix = strings.ReplaceAll(suffix, "-", "")
		suffix = strings.ReplaceAll(suffix, "+", "")
		suffix = strings.ReplaceAll(suffix, "/", "")
		suffix = strings.ReplaceAll(suffix, ".", "")
		suffix = strings.ReplaceAll(suffix, ":", "")
		result = append(result, element{
			Name: "Generated" + name + "_" + suffix,
			Doc:  newDock,
		})
	}
	return result
}

func replaceMethodNameInDoc(doc string, oldName string, name string) string {
	lines := strings.Split(doc, "\n")
	comments := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			comments = append(comments, "// "+strings.ReplaceAll(line, oldName+" godoc", name+" godoc"))
		}
	}
	return strings.Join(comments, "\n")
}

func updateEndpoint(doc string, prefix string) (newDoc string, method string, path string) {
	splitByRouter := strings.SplitN(doc, "@Router", 2)
	if len(splitByRouter) >= 2 {
		parts := strings.Split(splitByRouter[1], "/")
		if len(parts) >= 2 && strings.TrimSpace(parts[0]) == "" {
			parts = slices.Insert(parts, 1, prefix)
		} else {
			parts = slices.Insert(parts, 0, "/"+prefix)
		}
		splitByRouter[1] = strings.Join(parts, "/")
		method, path = parseEndpoint(splitByRouter[1])
	}
	newDoc = strings.Join(splitByRouter, "@Router")
	return newDoc, method, path
}

func parseEndpoint(s string) (method string, path string) {
	s = strings.TrimSpace(s)
	s = strings.Split(s, "]")[0]
	parts := strings.Split(s, "[")
	if len(parts) != 2 {
		panic(fmt.Sprintf("invalid endpoint format in '%v'", s))
	}
	method = strings.ToUpper(strings.TrimSpace(parts[1]))
	path = strings.TrimSpace(parts[0])
	return method, path
}

func filterMethods(dirAst map[string]*ast.Package) (result []*ast.FuncDecl) {
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
func {{$element.Name}}(){
	//this method is only used as anchor for swagger documentation
	panic("this method is only used as anchor for swagger documentation")
}


{{end}}
`
