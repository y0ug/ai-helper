package repomap

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	treesitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	treesitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

const goFunctionQuery = `
(function_declaration
    name: (identifier) @func_name
    parameters: (parameter_list) @params
    return_type: (type) @return_type)
`

const jsClassQuery = `
(class_declaration
    name: (identifier) @class_name
    body: (class_body
        (method_definition
            name: (property_identifier) @method_name
            parameters: (formal_parameters) @method_params
            return_type: (_) @method_return_type)))
`

type FunctionInfo struct {
	Name       string
	Parameters []Parameter
	ReturnType string
}

type Parameter struct {
	Name string
	Type string
}

type ClassInfo struct {
	Name    string
	Methods []FunctionInfo
}

type RepoMap struct {
	Classes   []ClassInfo
	Functions []FunctionInfo
	// Add Types, etc., as needed
}

func executeQuery(
	content []byte,
	lang tree_sitter.Language,
	queryStr string,
) ([]map[string]string, error) {
	parser := tree_sitter.NewParser()
	parser.SetLanguage(lang)
	tree := parser.Parse(content)
	query, err := tree_sitter.NewQuery(lang, queryStr)
	if err != nil {
		return nil, err
	}
	queryCursor := tree_sitter.NewQueryCursor()
	queryCursor.Exec(query, tree.RootNode())

	var results []map[string]string
	for {
		match, ok := queryCursor.NextMatch()
		if !ok {
			break
		}
		captures := make(map[string]string)
		for _, c := range match.Captures {
			captures[c.Name] = string(c.Node.Content(content))
		}
		results = append(results, captures)
	}
	return results, nil
}

func parseGoParameters(paramStr string) []Parameter {
	// Implement parsing logic for Go parameters
	// This can involve using Tree-sitter again or simple string parsing
	// For simplicity, here's a dummy implementation
	return []Parameter{
		{Name: "param1", Type: "int"},
		{Name: "param2", Type: "string"},
	}
}

func cloneRepo(url, directory string) (*git.Repository, error) {
	repo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func traverseRepo(
	root string,
	languageMap map[string]tree_sitter.Language,
	repoMap *RepoMap,
) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		lang, exists := languageMap[ext]
		if !exists {
			return nil // Skip unsupported file types
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			return nil
		}
		processFile(path, lang, content, repoMap)
		return nil
	})
}

func processFile(path string, lang tree_sitter.Language, content []byte, repoMap *RepoMap) {
	var queryStr string
	var language string

	switch lang.Type() {
	case treesitter_go.GetLanguage().Type():
		queryStr = goFunctionQuery
		language = "go"
	case treesitter_javascript.GetLanguage().Type():
		queryStr = jsClassQuery
		language = "javascript"
	default:
		return // Unsupported language
	}

	results, err := executeQuery(content, lang, queryStr)
	if err != nil {
		log.Printf("Query execution failed for file %s: %v", path, err)
		return
	}

	for _, res := range results {
		switch language {
		case "go":
			funcInfo := FunctionInfo{
				Name:       res["func_name"],
				Parameters: parseGoParameters(res["params"]),
				ReturnType: res["return_type"],
			}
			repoMap.Functions = append(repoMap.Functions, funcInfo)
		case "javascript":
			classInfo := ClassInfo{
				Name: res["class_name"],
				// You can similarly extract methods by executing another query or extending the current one
			}
			repoMap.Classes = append(repoMap.Classes, classInfo)
		}
	}
}
