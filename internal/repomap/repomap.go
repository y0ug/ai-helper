package repomap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

const goFunctionQuery = `
(
  (comment)* @doc
  .
  (function_declaration
    name: (identifier) @name.definition.function) @definition.function
  (#strip! @doc "^//\\s*")
  (#set-adjacent! @doc @definition.function)
)

(
  (comment)* @doc
  .
  (method_declaration
    name: (field_identifier) @name.definition.method) @definition.method
  (#strip! @doc "^//\\s*")
  (#set-adjacent! @doc @definition.method)
)

(call_expression
  function: [
    (identifier) @name.reference.call
    (parenthesized_expression (identifier) @name.reference.call)
    (selector_expression field: (field_identifier) @name.reference.call)
    (parenthesized_expression (selector_expression field: (field_identifier) @name.reference.call))
  ]) @reference.call

(type_spec
  name: (type_identifier) @name.definition.type) @definition.type

(type_identifier) @name.reference.type @reference.type
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
	sourceCode []byte,
	lang *tree_sitter.Language,
	queryStr string,
) ([]map[string]string, error) {
	parser := tree_sitter.NewParser()
	parser.SetLanguage(lang)
	defer parser.Close()

	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()

	query, err := tree_sitter.NewQuery(lang, queryStr)
	if err != nil {
		return nil, err
	}
	defer query.Close()

	qc := tree_sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), sourceCode)

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			fmt.Printf(
				"Match %d, Capture %d (%s): %s\n",
				match.PatternIndex,
				capture.Index,
				query.CaptureNames()[capture.Index],
				capture.Node.Utf8Text(sourceCode),
			)
		}
	}
	return nil, nil
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

// func traverseRepo(
//
//	    root string,
//	    languageMap map[string]*tree_sitter.Language,
//	    repoMap *RepoMap,
//	) error {
//	    // Open the repository
//	    repo, err := git.PlainOpen(root)
//	    if err != nil {
//	        return fmt.Errorf("failed to open repository: %w", err)
//	    }
//
//	    // Get the worktree
//	    wt, err := repo.Worktree()
//	    if err != nil {
//	        return fmt.Errorf("failed to get worktree: %w", err)
//	    }
//
//	    // Get the status to find tracked files
//	    status, err := wt.Status()
//	    if err != nil {
//	        return fmt.Errorf("failed to get repository status: %w", err)
//	    }
//
//	    // Process each tracked file
//	    for filePath, fileStatus := range status {
//	        // Skip untracked files
//	        if fileStatus.Worktree == git.Untracked {
//	            continue
//	        }
//
//	        // Check file extension
//	        ext := filepath.Ext(filePath)
//	        lang, exists := languageMap[ext]
//	        if !exists {
//	            continue // Skip unsupported file types
//	        }
//
//	        // Read file content
//	        fullPath := filepath.Join(root, filePath)
//	        content, err := ioutil.ReadFile(fullPath)
//	        if err != nil {
//	            log.Printf("Failed to read file %s: %v", fullPath, err)
//	            continue
//	        }
//
//	        processFile(fullPath, lang, content, repoMap)
//	    }
//
//	    return nil
//	}
func traverseRepo(
	root string,
	languageMap map[string]*tree_sitter.Language,
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
		fmt.Println(ext)
		lang, exists := languageMap[ext]
		if !exists {
			return nil // Skip unsupported file types
		}
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file %s: %v", path, err)
			return nil
		}
		fmt.Println(path)
		processFile(path, ext, lang, content, repoMap)
		return nil
	})
}

func processFile(
	path string,
	ext string,
	lang *tree_sitter.Language,
	content []byte,
	repoMap *RepoMap,
) {
	var queryStr string
	switch ext {
	case ".go":
		queryStr = goFunctionQuery
	case ".js":
		queryStr = jsClassQuery
	default:
		return
	}
	results, err := executeQuery(content, lang, queryStr)
	if err != nil {
		log.Printf("Query execution failed for file %s: %v", path, err)
		return
	}

	for _, res := range results {
		switch ext {
		case ".go":
			funcInfo := FunctionInfo{
				Name:       res["func_name"],
				Parameters: parseGoParameters(res["params"]),
				ReturnType: res["return_type"],
			}
			repoMap.Functions = append(repoMap.Functions, funcInfo)
		case ".js":
			classInfo := ClassInfo{
				Name: res["class_name"],
				// You can similarly extract methods by executing another query or extending the current one
			}
			repoMap.Classes = append(repoMap.Classes, classInfo)
		}
	}
}
