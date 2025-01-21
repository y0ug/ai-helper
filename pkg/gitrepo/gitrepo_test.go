package gitrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Helper function to create a test repository
func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	// Create basic git config
	cfg, err := repo.Config()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}
	cfg.User.Name = "test"
	cfg.User.Email = "test@test.com"
	repo.SetConfig(cfg)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func createTestFile(t *testing.T, path string, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	assert.NoError(t, err)
}

func TestCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	mockIO := NewMockIOInterface(ctrl)

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	createTestFile(t, testFile, "test content")

	tests := []struct {
		name       string
		fnames     []string
		context    string
		message    string
		aiderEdits bool
		setupMocks func(*MockIOInterface)
		wantErr    bool
	}{
		{
			name:       "successful commit",
			fnames:     []string{"test.txt"},
			context:    "test context",
			message:    "test commit",
			aiderEdits: true,
			setupMocks: func(m *MockIOInterface) {
				m.EXPECT().ToolError(gomock.Any()).AnyTimes() // Allow any error messages
				m.EXPECT().ToolOutput(gomock.Any(), true).Times(1)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks(mockIO)
			}

			config := Config{
				IO:              mockIO,
				GitDname:        repoPath,
				AttributeAuthor: true,
			}

			gr, err := New(config)
			assert.NoError(t, err)

			// Stage the files before commit
			w, err := gr.repo.Worktree()
			assert.NoError(t, err)
			_, err = w.Add("test.txt")
			assert.NoError(t, err)

			hash, msg, err := gr.Commit(tt.fnames, tt.context, tt.message, tt.aiderEdits)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.Equal(t, tt.message, msg)
		})
	}
}

func TestGetDiffs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	mockIO := NewMockIOInterface(ctrl)

	config := Config{
		IO:       mockIO,
		GitDname: repoPath,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	createTestFile(t, testFile, "test content")

	// Add the file to git
	w, err := gr.repo.Worktree()
	assert.NoError(t, err)
	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	diffs, err := gr.GetDiffs([]string{"test.txt"})
	assert.NoError(t, err)
	assert.Contains(t, diffs, "test.txt")
	assert.Contains(t, diffs, "test content") // Should contain the added content
}

func TestGetTrackedFiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	mockIO := NewMockIOInterface(ctrl)

	config := Config{
		IO:       mockIO,
		GitDname: repoPath,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	// Create and add a test file
	testFile := filepath.Join(repoPath, "test.txt")
	createTestFile(t, testFile, "test content")

	w, err := gr.repo.Worktree()
	assert.NoError(t, err)
	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
		},
	})
	assert.NoError(t, err)

	files, err := gr.GetTrackedFiles()
	assert.NoError(t, err)
	assert.Contains(t, files, "test.txt")
}

func TestIsIgnoredFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	mockIO := NewMockIOInterface(ctrl)

	// Create .gitignore
	gitignoreFile := filepath.Join(repoPath, ".gitignore")
	createTestFile(t, gitignoreFile, "*.log\nnode_modules/")

	config := Config{
		IO:       mockIO,
		GitDname: repoPath,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	assert.True(t, gr.IsIgnoredFile("test.log"))
	assert.True(t, gr.IsIgnoredFile("node_modules/test.js"))
	assert.False(t, gr.IsIgnoredFile("test.txt"))
}

func TestGetHeadCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	mockIO := NewMockIOInterface(ctrl)

	config := Config{
		IO:       mockIO,
		GitDname: repoPath,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
	createTestFile(t, testFile, "test content")

	w, err := gr.repo.Worktree()
	assert.NoError(t, err)
	_, err = w.Add("test.txt")
	assert.NoError(t, err)

	expectedHash, err := w.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
		},
	})
	assert.NoError(t, err)

	hash, err := gr.GetHeadCommit()
	assert.NoError(t, err)
	assert.Equal(t, expectedHash.String(), hash)
}

// Real IO implementation for testing
type TestIO struct {
	errors   []string
	outputs  []string
	warnings []string
}

func (io *TestIO) ToolError(msg string) {
	io.errors = append(io.errors, msg)
}

func (io *TestIO) ToolOutput(msg string, bold bool) {
	io.outputs = append(io.outputs, msg)
}

func (io *TestIO) ToolWarning(msg string) {
	io.warnings = append(io.warnings, msg)
}

func TestBasicGitOperations(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	testIO := &TestIO{}
	config := Config{
		IO:                              testIO,
		GitDname:                        repoPath,
		AttributeAuthor:                 true,
		AttributeCommitter:              true,
		AttributeCommitMessageAuthor:    true,
		AttributeCommitMessageCommitter: true,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	t.Run("commit new file", func(t *testing.T) {
		// Create and add a test file
		testFile := filepath.Join(repoPath, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		assert.NoError(t, err)

		// Add file to git
		w, err := gr.repo.Worktree()
		assert.NoError(t, err)
		hash2, err := w.Add("test.txt")
		assert.NoError(t, err)
		fmt.Println(hash2)
		// Commit the file
		hash, msg, err := gr.Commit([]string{"test.txt"}, "initial commit", "Add test file", true)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Equal(t, "Add test file", msg)

		// Check if the commit was recorded in output
		found := false
		for _, output := range testIO.outputs {
			if strings.Contains(output, "Add test file") {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	// Add more test cases...
}

func TestEdgeCases(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	testIO := &TestIO{}
	config := Config{
		IO:       testIO,
		GitDname: repoPath,
	}

	gr, err := New(config)
	assert.NoError(t, err)

	t.Run("non-existent file", func(t *testing.T) {
		assert.False(t, gr.PathInRepo("non-existent.txt"))

		hash, msg, err := gr.Commit(
			[]string{"non-existent.txt"},
			"",
			"Try commit non-existent",
			false,
		)
		assert.Empty(t, hash)
		assert.Empty(t, msg)
		assert.NoError(t, err)
	})

	t.Run("empty commit", func(t *testing.T) {
		hash, msg, err := gr.Commit(nil, "", "", false)
		assert.Empty(t, hash)
		assert.Empty(t, msg)
		assert.NoError(t, err)
	})
}

// Add more test functions for other operations...
