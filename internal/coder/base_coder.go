package coder

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/pkg/gitrepo"
)

type Edit struct {
	Path         string
	OriginalText string
	UpdatedText  string
}

// Fence represents a pair of opening and closing delimiters for code blocks
type Fence [2]string

// DefaultFences defines all possible fencing options in order of preference
var DefaultFences = []Fence{
	{"```", "```"},
	{"````", "````"},
	{"<source>", "</source>"},
	{"<code>", "</code>"},
	{"<pre>", "</pre>"},
	{"<codeblock>", "</codeblock>"},
	{"<sourcecode>", "</sourcecode>"},
}

type BaseCoder struct {
	mainModel  *models.Model
	editFormat string
	// io                *io.InputOutput
	repo                 *gitrepo.GitRepo
	curMessages          []prompts.Message
	doneMessages         []prompts.Message
	absFileNames         map[string]struct{}
	absReadOnlyNames     map[string]struct{}
	lastCommitHash       string
	aiderCommitHashes    map[string]struct{}
	temperature          float64
	autoLint             bool
	autoTest             bool
	testCmd              string
	totalCost            float64
	chatLanguage         string
	verbose              bool
	prompts              prompts.BasePrompts
	fence                Fence
	root                 string
	absRootPathCache     map[string]string
	lintCommands         map[string]string
	suggestShellCommands bool
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	return &BaseCoder{
		// mainModel:  opts.MainModel,
		// editFormat: opts.EditFormat,
		// // io:                opts.IO,
		// // repo:              opts.Repo,
		// absFileNames:      make(map[string]struct{}),
		// absReadOnlyNames:  make(map[string]struct{}),
		// aiderCommitHashes: make(map[string]struct{}),
		prompts: *prompts.NewBasePrompts(),
	}
}

func (c *BaseCoder) getPrompts() *prompts.BasePrompts {
	return &c.prompts
}

func (c *BaseCoder) InitBeforeMessage() {
	// Reset state before processing a new message
	// if c.repo != nil {
	// 	c.lastCommitHash = c.repo.GetHeadCommitSHA()
	// }
}

func (c *BaseCoder) Run(message string) error {
	c.InitBeforeMessage()
	return c.SendMessage(message)
}

func (c *BaseCoder) SendMessage(message string) error {
	// Add user message
	c.curMessages = append(c.curMessages, prompts.Message{
		Role:    "user",
		Content: message,
	})

	// Format messages with appropriate prompts
	_ = c.FormatMessages()

	// // Send to LLM and handle response
	// response, err := c.SendToLLM(messages)
	// if err != nil {
	// 	return err
	// }
	//
	// // Process response and apply edits
	// edits, err := c.GetEdits(response)
	// if err != nil {
	// 	return err
	// }
	//
	// return c.ApplyEdits(edits)
	//
	return nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	c.chooseFence()
	chunks := &ChatChunks{}

	// Add system messages
	systemPrompt := c.formatSystemPrompt()
	hasSystemPrompt := true
	if hasSystemPrompt {
		chunks.System = []prompts.Message{{
			Role:    "system",
			Content: systemPrompt,
		}}
	} else {
		chunks.System = []prompts.Message{
			{Role: "user", Content: systemPrompt},
			{Role: "assistant", Content: "Ok."},
		}
	}

	// Add example messages from prompts
	chunks.Examples = c.getPrompts().ExampleMessages

	// Add chat history
	chunks.Done = c.doneMessages

	// Add repo content if available
	if repoMsgs := c.getRepoMessages(); len(repoMsgs) > 0 {
		chunks.Repo = repoMsgs
	}

	// Add readonly files content
	if readOnlyMsgs := c.getReadOnlyFilesMessages(); len(readOnlyMsgs) > 0 {
		chunks.ReadOnlyFiles = readOnlyMsgs
	}

	// Add chat files content
	if chatFilesMsgs := c.getChatFilesMessages(); len(chatFilesMsgs) > 0 {
		chunks.ChatFiles = chatFilesMsgs
	}

	// Add current conversation
	chunks.Cur = c.curMessages

	// Add reminder if needed
	if reminder := c.getPrompts().SystemReminder; reminder != "" {
		chunks.Reminder = []prompts.Message{{
			Role:    "system",
			Content: reminder,
		}}
	}

	return chunks
}

func (c *BaseCoder) getRepoMessages() []prompts.Message {
	// if c.repo == nil {
	// 	return nil
	// }
	//
	// repoContent := c.getRepoMap()
	// if repoContent == "" {
	// 	return nil
	// }
	//
	// return []prompts.Message{
	// 	{
	// 		Role:    "user",
	// 		Content: repoContent,
	// 	},
	// 	{
	// 		Role:    "assistant",
	// 		Content: "Ok, I won't try and edit those files without asking first.",
	// 	},
	// }
	return []prompts.Message{}
}

func (c *BaseCoder) getReadOnlyFilesMessages() []prompts.Message {
	if len(c.absReadOnlyNames) == 0 {
		return nil
	}

	content := c.getReadOnlyFilesContent()
	if content == "" {
		return nil
	}

	promptsR := c.getPrompts()
	return []prompts.Message{
		{
			Role:    "user",
			Content: promptsR.ReadOnlyFilesPrefix + "\n" + content,
		},
		{
			Role:    "assistant",
			Content: "Ok, I will use these files as references.",
		},
	}
}

func (c *BaseCoder) getRepoMap() string {
	// if c.repo == nil {
	//   return ""
	// }
	// return c.repo.GetRepoMap()
	return ""
}

func (c *BaseCoder) getChatFilesMessages() []prompts.Message {
	if len(c.absFileNames) == 0 {
		promptsR := c.getPrompts()
		if c.getRepoMap() != "" && promptsR.FilesNoFullFilesWithRepoMap != "" {
			return []prompts.Message{
				{Role: "user", Content: promptsR.FilesNoFullFilesWithRepoMap},
				{Role: "assistant", Content: promptsR.FilesNoFullFilesWithRepoMapReply},
			}
		}
		return []prompts.Message{
			{Role: "user", Content: promptsR.FilesNoFullFiles},
			{Role: "assistant", Content: "Ok."},
		}
	}

	promptsR := c.getPrompts()
	content := promptsR.FilesContentPrefix + "\n" + c.getFilesContent()

	return []prompts.Message{
		{Role: "user", Content: content},
		{Role: "assistant", Content: promptsR.FilesContentAssistantReply},
	}
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (c *BaseCoder) getFilesContent() string {
	var content string
	for fname := range c.absFileNames {
		relPath := c.getRelativePath(fname)
		fileContent, err := readFile(fname)
		if err != nil {
			continue
		}

		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += string(fileContent)
		content += c.fence[1] + "\n"
	}
	return content
}

func (c *BaseCoder) getReadOnlyFilesContent() string {
	var content string
	for fname := range c.absReadOnlyNames {
		relPath := c.getRelativePath(fname)
		fileContent, err := readFile(fname)
		if err != nil {
			continue
		}

		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += string(fileContent)
		content += c.fence[1] + "\n"
	}
	return content
}

func (c *BaseCoder) formatSystemPrompt() string {
	promptsR := c.getPrompts()
	data := TemplateData{
		Language:       c.getLanguage(),
		LazyPrompt:     c.getLazyPrompt(),
		Platform:       c.getPlatformInfo(),
		ShellCmdPrompt: c.getShellCmdPrompt(),
		Fence:          c.fence,
	}

	formatted, err := prompts.RenderTemplate(promptsR.MainSystem, data)
	if err != nil {
		return promptsR.MainSystem
	}
	return formatted
}

func (c *BaseCoder) getLanguage() string {
	if c.chatLanguage != "" {
		return c.chatLanguage
	}
	return "the same language they are using"
}

func (c *BaseCoder) getLazyPrompt() string {
	if c.mainModel.Lazy {
		return c.getPrompts().LazyPrompt
	}
	return ""
}

type TemplateData struct {
	Language       string
	LazyPrompt     string
	Platform       string
	ShellCmdPrompt string
	Fence          [2]string
}

// chooseFence selects appropriate fence markers that won't conflict with file contents
func (c *BaseCoder) chooseFence() {
	// Get all content from files to check for fence conflicts
	allContent := c.getAllContent()

	// Try each fence option until we find one that doesn't appear in the content
	for _, fence := range DefaultFences {
		if !hasFenceConflict(allContent, fence) {
			c.fence = fence
			return
		}
	}

	// If all fences conflict (unlikely), use the default and warn
	c.fence = DefaultFences[0]
	// c.io.ToolWarning("Unable to find a non-conflicting fence strategy! Falling back to: " +
	// 	c.fence[0] + "..." + c.fence[1])
}

// getAllContent combines content from all files being handled
func (c *BaseCoder) getAllContent() string {
	var builder strings.Builder

	// Get content from editable files
	for fname := range c.absFileNames {
		content, err := readFile(fname)
		if err == nil {
			builder.WriteString(string(content))
			builder.WriteString("\n")
		}
	}

	// Get content from read-only files
	for fname := range c.absReadOnlyNames {
		content, err := readFile(fname)
		if err == nil {
			builder.WriteString(string(content))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// hasFenceConflict checks if fence markers appear in the content
func hasFenceConflict(content string, fence Fence) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, fence[0]) || strings.HasPrefix(line, fence[1]) {
			return true
		}
	}
	return false
}

// getRelativePath converts an absolute path to a path relative to the repository root or working directory
func (c *BaseCoder) getRelativePath(absPath string) string {
	// Check cache first
	if relPath, ok := c.absRootPathCache[absPath]; ok {
		return relPath
	}

	// Get relative path
	relPath, err := filepath.Rel(c.root, absPath)
	if err != nil {
		// If we can't get relative path, return absolute path
		return absPath
	}

	// Cache and return the result
	c.absRootPathCache[absPath] = relPath
	return relPath
}

// absRootPath converts a relative path to absolute path using root directory
func (c *BaseCoder) absRootPath(path string) string {
	// Check cache first
	if cached, ok := c.absRootPathCache[path]; ok {
		return cached
	}

	// Join with root and get absolute path
	absPath := filepath.Join(c.root, path)
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		// If we can't get absolute path, return joined path
		absPath = filepath.Join(c.root, path)
	}

	// Cache and return result
	c.absRootPathCache[path] = absPath
	return absPath
}
