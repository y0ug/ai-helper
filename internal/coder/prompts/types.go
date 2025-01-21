package prompts

// BasePrompts represents the base prompts configuration
type BasePrompts struct {
	SystemReminder                   string    `yaml:"system_reminder"`
	FilesContentGPTEdits             string    `yaml:"files_content_gpt_edits"`
	FilesContentGPTEditsNoRepo       string    `yaml:"files_content_gpt_edits_no_repo"`
	FilesContentGPTNoEdits           string    `yaml:"files_content_gpt_no_edits"`
	FilesContentLocalEdits           string    `yaml:"files_content_local_edits"`
	LazyPrompt                       string    `yaml:"lazy_prompt"`
	ExampleMessages                  []Message `yaml:"example_messages"`
	FilesContentPrefix               string    `yaml:"files_content_prefix"`
	FilesContentAssistantReply       string    `yaml:"files_content_assistant_reply"`
	FilesNoFullFiles                 string    `yaml:"files_no_full_files"`
	FilesNoFullFilesWithRepoMap      string    `yaml:"files_no_full_files_with_repo_map"`
	FilesNoFullFilesWithRepoMapReply string    `yaml:"files_no_full_files_with_repo_map_reply"`
	RepoContentPrefix                string    `yaml:"repo_content_prefix"`
	ReadOnlyFilesPrefix              string    `yaml:"read_only_files_prefix"`
	ShellCmdPrompt                   string    `yaml:"shell_cmd_prompt"`
	ShellCmdReminder                 string    `yaml:"shell_cmd_reminder"`
	NoShellCmdPrompt                 string    `yaml:"no_shell_cmd_prompt"`
	NoShellCmdReminder               string    `yaml:"no_shell_cmd_reminder"`
}

type BasePromptsConfig struct {
	BasePrompts *BasePrompts `yaml:"base_prompts"`
}

// Message represents a chat message
type Message struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

// ArchitectPrompts represents the architect-specific prompts
type ArchitectPrompts struct {
	BasePrompts
	MainSystem string `yaml:"main_system"`
}

// AskPrompts represents the ask-specific prompts
type AskPrompts struct {
	BasePrompts
	MainSystem string `yaml:"main_system"`
}

// EditBlockPrompts represents the editblock-specific prompts
type EditBlockPrompts struct {
	BasePrompts
	MainSystem string `yaml:"main_system"`
}

// EditBlockFencedPrompts represents the editblock-fenced-specific prompts
type EditBlockFencedPrompts struct {
	EditBlockPrompts
}

// EditBlockFunctionPrompts represents the editblock-function-specific prompts
type EditBlockFunctionPrompts struct {
	BasePrompts
	MainSystem          string `yaml:"main_system"`
	RedactedEditMessage string `yaml:"redacted_edit_message"`
}

// EditorEditBlockPrompts represents the editor-editblock-specific prompts
type EditorEditBlockPrompts struct {
	EditBlockPrompts
	MainSystem string `yaml:"main_system"`
}

// EditorWholeFilePrompts represents the editor-wholefile-specific prompts
type EditorWholeFilePrompts struct {
	WholeFilePrompts
	MainSystem string `yaml:"main_system"`
}

// HelpPrompts represents the help-specific prompts
type HelpPrompts struct {
	BasePrompts
	MainSystem string `yaml:"main_system"`
}

// SingleWholeFileFunctionPrompts represents the single-wholefile-function-specific prompts
type SingleWholeFileFunctionPrompts struct {
	BasePrompts
	MainSystem          string `yaml:"main_system"`
	SystemReminder      string `yaml:"system_reminder"`
	RedactedEditMessage string `yaml:"redacted_edit_message"`
}

// WholeFilePrompts represents the wholefile-specific prompts
type WholeFilePrompts struct {
	BasePrompts
	MainSystem          string `yaml:"main_system"`
	SystemReminder      string `yaml:"system_reminder"`
	RedactedEditMessage string `yaml:"redacted_edit_message"`
}

// PromptsConfig contains all prompt configurations
type PromptsConfig struct {
	BasePrompts                    *BasePrompts                    `yaml:"base_prompts"`
	ArchitectPrompts               *ArchitectPrompts               `yaml:"architect_prompts"`
	AskPrompts                     *AskPrompts                     `yaml:"ask_prompts"`
	EditBlockPrompts               *EditBlockPrompts               `yaml:"editblock_prompts"`
	EditBlockFencedPrompts         *EditBlockFencedPrompts         `yaml:"editblock_fenced_prompts"`
	EditBlockFunctionPrompts       *EditBlockFunctionPrompts       `yaml:"editblock_func_prompts"`
	EditorEditBlockPrompts         *EditorEditBlockPrompts         `yaml:"editor_editblock_prompts"`
	EditorWholeFilePrompts         *EditorWholeFilePrompts         `yaml:"editor_wholefile_prompts"`
	HelpPrompts                    *HelpPrompts                    `yaml:"help_prompts"`
	SingleWholeFileFunctionPrompts *SingleWholeFileFunctionPrompts `yaml:"single_wholefile_func_prompts"`
	WholeFilePrompts               *WholeFilePrompts               `yaml:"wholefile_prompts"`
}

// TemplateData contains data for template rendering
type TemplateData struct {
	Language       string
	Hash           string
	Message        string
	Platform       string
	LazyPrompt     string
	ShellCmdPrompt string
	Fence0         string
	Fence1         string
}

// YAMLConfig represents a YAML file with imports
type YAMLConfig struct {
	Imports []string    `yaml:"imports,omitempty"`
	Data    interface{} `yaml:",inline"`
}
