package llmcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRequestContext(t *testing.T) {
	input := "test input"
	rc := NewRequestContext(input)

	if rc.Input != input {
		t.Errorf("Expected input %q, got %q", input, rc.Input)
	}

	if rc.Env == nil {
		t.Error("Environment map should be initialized")
	}

	if rc.Files == nil {
		t.Error("Files map should be initialized")
	}

	if rc.Vars == nil {
		t.Error("Vars map should be initialized")
	}
}

func TestLoadEnvironment(t *testing.T) {
	// Set up test environment variables
	testEnvKey := "TEST_ENV_VAR"
	testEnvValue := "test_value"
	os.Setenv(testEnvKey, testEnvValue)
	defer os.Unsetenv(testEnvKey)

	rc := NewRequestContext("")
	rc.LoadEnvironment()

	if value, exists := rc.Env[testEnvKey]; !exists {
		t.Errorf("Environment variable %s not loaded", testEnvKey)
	} else if value != testEnvValue {
		t.Errorf("Expected env value %q, got %q", testEnvValue, value)
	}
}

func TestLoadFiles(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()
	
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testContent1 := "test content 1"
	if err := os.WriteFile(testFile1, []byte(testContent1), 0644); err != nil {
		t.Fatal(err)
	}

	testFile2 := filepath.Join(tmpDir, "test2.txt")
	testContent2 := "test content 2"
	if err := os.WriteFile(testFile2, []byte(testContent2), 0644); err != nil {
		t.Fatal(err)
	}

	rc := NewRequestContext("")
	err := rc.LoadFiles([]string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("LoadFiles failed: %v", err)
	}

	if content, exists := rc.Files[testFile1]; !exists {
		t.Errorf("File %s not loaded", testFile1)
	} else if content != testContent1 {
		t.Errorf("Expected content %q, got %q", testContent1, content)
	}

	if content, exists := rc.Files[testFile2]; !exists {
		t.Errorf("File %s not loaded", testFile2)
	} else if content != testContent2 {
		t.Errorf("Expected content %q, got %q", testContent2, content)
	}

	// Test loading non-existent file
	err = rc.LoadFiles([]string{"nonexistent.txt"})
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

func TestTemplateFuncs(t *testing.T) {
	rc := NewRequestContext("")
	rc.Files["test.txt"] = "test content"
	funcs := GetTemplateFuncs(rc)

	// Test fileContent
	fileContentFunc := funcs["fileContent"].(func(string) string)
	content := fileContentFunc("test.txt")
	if content != "test content" {
		t.Errorf("Expected content %q, got %q", "test content", content)
	}

	// Test non-existent file
	content = fileContentFunc("nonexistent.txt")
	if content != "Error: file nonexistent.txt not found" {
		t.Errorf("Expected error message for non-existent file, got %q", content)
	}

	// Test fileExt
	fileExtFunc := funcs["fileExt"].(func(string) string)
	ext := fileExtFunc("test.txt")
	if ext != ".txt" {
		t.Errorf("Expected extension %q, got %q", ".txt", ext)
	}

	// Test fileName
	fileNameFunc := funcs["fileName"].(func(string) string)
	name := fileNameFunc("/path/to/test.txt")
	if name != "test.txt" {
		t.Errorf("Expected filename %q, got %q", "test.txt", name)
	}

	// Test formatFile
	formatFileFunc := funcs["formatFile"].(func(string) string)
	formatted := formatFileFunc("test.txt")
	expected := "```txt\ntest content\n```"
	if formatted != expected {
		t.Errorf("Expected formatted content %q, got %q", expected, formatted)
	}
}

func TestExecute(t *testing.T) {
	rc := NewRequestContext("test input")
	rc.Files["test.txt"] = "file content"
	rc.Vars["testVar"] = "variable content"

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "basic template",
			template: "Input: {{.Input}}",
			want:     "Input: test input",
			wantErr:  false,
		},
		{
			name:     "file content",
			template: "File: {{fileContent \"test.txt\"}}",
			want:     "File: file content",
			wantErr:  false,
		},
		{
			name:     "formatted file",
			template: "{{formatFile \"test.txt\"}}",
			want:     "```txt\nfile content\n```",
			wantErr:  false,
		},
		{
			name:     "variable",
			template: "Var: {{.Vars.testVar}}",
			want:     "Var: variable content",
			wantErr:  false,
		},
		{
			name:     "invalid template",
			template: "{{.InvalidField}}",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Execute(tt.template, rc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
