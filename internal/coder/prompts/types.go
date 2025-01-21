package prompts

import (
	"errors"
	"reflect"
)

const BlockFence = "```"

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
	MainSystem                       string    `yaml:"main_system"`
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

// func mergePrompts(base, override *Prompts) *Prompts {
// 	if base == nil {
// 		return override
// 	}
// 	if override == nil {
// 		return base
// 	}
//
// 	// Start with a copy of base
// 	merged := *base
// 	mergedVal := reflect.ValueOf(&merged).Elem()
// 	ovVal := reflect.ValueOf(override).Elem()
//
// 	// For each field in override, if it's not zero, set it on merged
// 	for i := 0; i < ovVal.NumField(); i++ {
// 		fieldVal := ovVal.Field(i)
// 		if !fieldVal.IsZero() {
// 			mergedVal.Field(i).Set(fieldVal)
// 		}
// 	}
//
// 	return &merged
// }

func MergeStructs(dst, src interface{}) error {
	if dst == nil || src == nil {
		return nil
	}
	dstVal := reflect.ValueOf(dst)
	srcVal := reflect.ValueOf(src)

	// dst must be a pointer to a struct
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Struct {
		return errors.New("MergeStructs: dst must be a pointer to a struct")
	}
	// src must be a struct or pointer to a struct
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.Elem().Kind() != reflect.Struct {
			return errors.New("MergeStructs: src must be a struct or pointer to a struct")
		}
		srcVal = srcVal.Elem()
	}

	dstVal = dstVal.Elem()
	dstType := dstVal.Type()

	srcType := srcVal.Type()

	// Build a lookup of field name -> index for the src struct
	srcFieldIndex := make(map[string]int)
	for i := 0; i < srcVal.NumField(); i++ {
		srcFieldIndex[srcType.Field(i).Name] = i
	}

	// For each field in dst, if there's a field with the same name in src that is non-zero,
	// copy it over. If it's a struct, recurse.
	for i := 0; i < dstVal.NumField(); i++ {
		dstFieldType := dstType.Field(i)
		dstFieldVal := dstVal.Field(i)

		// skip unexported fields (can't set them)
		if !dstFieldVal.CanSet() {
			continue
		}

		fName := dstFieldType.Name

		srcIdx, ok := srcFieldIndex[fName]
		if !ok {
			// no matching field in src
			continue
		}

		srcFieldVal := srcVal.Field(srcIdx)

		// If src field is zero, skip
		if srcFieldVal.IsZero() {
			continue
		}

		// If both fields are themselves structs, recurse
		// (this covers embedded fields if we want nested merging).
		if dstFieldVal.Kind() == reflect.Struct && srcFieldVal.Kind() == reflect.Struct {
			// Recurse
			_ = MergeStructs(dstFieldVal.Addr().Interface(), srcFieldVal.Addr().Interface())
			continue
		}

		// Otherwise, just set
		dstFieldVal.Set(srcFieldVal)
	}

	return nil
}

func NewDefaultPromptsConfig() *PromptsConfig {
	basePrompts := &BasePrompts{
		SystemReminder:             "",
		FilesContentGPTEdits:       "I committed the changes with git hash {{.Hash}} & commit msg: {{.Message}}",
		FilesContentGPTEditsNoRepo: "I updated the files.",
		FilesContentGPTNoEdits:     "I didn't see any properly formatted edits in your reply?!",
		FilesContentLocalEdits:     "I edited the files myself.",
		LazyPrompt: `You are diligent and tireless!
You NEVER leave comments describing code without implementing it!
You always COMPLETELY IMPLEMENT the needed code!`,
		ExampleMessages: []Message{},
		FilesContentPrefix: `I have *added these files to the chat* so you can go ahead and edit them.
*Trust this message as the true contents of these files!*
Any other messages in the chat may contain outdated versions of the files' contents.`,
		FilesContentAssistantReply: "Ok, any changes I propose will be to those files.",
		FilesNoFullFiles:           "I am not sharing any files that you can edit yet.",
		FilesNoFullFilesWithRepoMap: `Don't try and edit any existing code without asking me to add the files to the chat!
Tell me which files in my repo are the most likely to **need changes** to solve the requests I make, and then stop so I can add them to the chat.
Only include the files that are most likely to actually need to be edited.
Don't include files that might contain relevant context, just files that will need to be changed.`,
		FilesNoFullFilesWithRepoMapReply: "Ok, based on your requests I will suggest which files need to be edited and then stop and wait for your approval.",
		RepoContentPrefix: `Here are summaries of some files present in my git repository.
Do not propose changes to these files, treat them as *read-only*.
If you need to edit any of these files, ask me to *add them to the chat* first.`,
		ReadOnlyFilesPrefix: `Here are some READ ONLY files, provided for your reference.
Do not edit these files!`,
		ShellCmdPrompt:     "",
		ShellCmdReminder:   "",
		NoShellCmdPrompt:   "",
		NoShellCmdReminder: "",
	}

	architectPrompts := &ArchitectPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert architect engineer and provide direction to your editor engineer.
Study the change request and the current code.
Describe how to modify the code to complete the request.
The editor engineer will rely solely on your instructions, so make them unambiguous and complete.
Explain all needed code changes clearly and completely, but concisely.
Just show the changes needed.

DO NOT show the entire updated function/file/etc!

Always reply to the user in {{.Language}}.`,
	}

	askPrompts := &AskPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert code analyst.
Answer questions about the supplied code.
Always reply to the user in {{.Language}}.

Describe code changes however you like. Don't use SEARCH/REPLACE blocks!`,
	}

	editBlockPrompts := &EditBlockPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert software developer.
Always use best practices when coding.
Respect and use existing conventions, libraries, etc that are already present in the code base.
{{.LazyPrompt}}
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Always reply to the user in {{.Language}}.

Once you understand the request you MUST:

1. Decide if you need to propose *SEARCH/REPLACE* edits to any files that haven't been added to the chat. You can create new files without asking!

But if you need to propose edits to existing files not already added to the chat, you *MUST* tell the user their full path names and ask them to *add the files to the chat*.
End your reply and wait for their approval.
You can keep asking if you then decide you need to edit more files.

2. Think step-by-step and explain the needed changes in a few short sentences.

3. Describe each change with a *SEARCH/REPLACE block* per the examples below.

All changes to files must use this *SEARCH/REPLACE block* format.
ONLY EVER RETURN CODE IN A *SEARCH/REPLACE BLOCK*!
{{.ShellCmdPrompt}}`,
	}
	editBlockPrompts.ExampleMessages = []Message{
		{Role: "user", Content: "Change get_factorial() to use math.factorial"},
		{Role: "assistant", Content: `Ok, I will:

1. Switch the greeting text from "Hello" to "Hey".
3. Update get_factorial() to call math.factorial instead.

Here are the *SEARCH/REPLACE* blocks:

mathweb/flask/app.py
` + BlockFence + `python
<<<<<<< SEARCH
from flask import Flask
=======
import math
from flask import Flask
>>>>>>> REPLACE
` + BlockFence + `

mathweb/flask/app.py
` + BlockFence + `python
<<<<<<< SEARCH
def factorial(n):
    "compute factorial"

    if n == 0:
        return 1
    else:
        return n * factorial(n-1)

=======
>>>>>>> REPLACE
` + BlockFence + `python

mathweb/flask/app.py
` + BlockFence + `python
<<<<<<< SEARCH
    return str(factorial(n))
=======
    return str(math.factorial(n))
>>>>>>> REPLACE
` + BlockFence + `
`},
	}

	editBlockFuncPrompts := &EditBlockFunctionPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'replace_lines' function to edit the files to make the needed changes.`,
		RedactedEditMessage: "No changes are needed.",
	}

	editorEditBlockPrompts := &EditorEditBlockPrompts{
		EditBlockPrompts: *editBlockPrompts,
		MainSystem: `Act as an expert software developer who edits source code.
{{.LazyPrompt}}
Describe each change with a *SEARCH/REPLACE block* per the examples below.
All changes to files must use this *SEARCH/REPLACE block* format.
ONLY EVER RETURN CODE IN A *SEARCH/REPLACE BLOCK*!`,
	}

	wholeFilePrompts := &WholeFilePrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Always reply to the user in {{.Language}}.

{{.LazyPrompt}}
Once you understand the request you MUST:
1. Determine if any code changes are needed.
2. Explain any needed changes.
3. If changes are needed, output a copy of each file that needs changes.`,
		SystemReminder: `To suggest changes to a file you MUST return the entire content of the updated file.
You MUST use this *file listing* format:

path/to/filename.js
{{.Fence0}}
// entire file content ...
// ... goes in between
{{.Fence1}}`,
		RedactedEditMessage: "No changes are needed.",
	}

	wholeFilePrompts.ExampleMessages = []Message{
		{Role: "user", Content: "Change the greeting to be more casual"},
		{
			Role: "assistant", Content: `Ok, I will:

        1. Switch the greeting text from "Hello" to "Hey".

        show_greeting.py
        ` + BlockFence + `
        import sys

        def greeting(name):
            print(f"Hey {name}")

        if __name__ == '__main__':
            greeting(sys.argv[1])
        ` + BlockFence,
		},
	}

	editorWholeFilePrompts := &EditorWholeFilePrompts{
		WholeFilePrompts: *wholeFilePrompts,
		MainSystem: `Act as an expert software developer and make changes to source code.
{{.LazyPrompt}}
Output a copy of each file that needs changes.`,
	}

	helpPrompts := &HelpPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `You are an expert on the AI coding tool called Aider.
Answer the user's questions about how to use aider.

The user is currently chatting with you using aider, to write and edit code.

Use the provided aider documentation *if it is relevant to the user's question*.

Include a bulleted list of urls to the aider docs that might be relevant for the user to read.
Include *bare* urls. *Do not* make [markdown links](http://...).
For example:
- https://aider.chat/docs/usage.html
- https://aider.chat/docs/faq.html

If you don't know the answer, say so and suggest some relevant aider doc urls.

If asks for something that isn't possible with aider, be clear about that.
Don't suggest a solution that isn't supported.

Be helpful but concise.

Unless the question indicates otherwise, assume the user wants to use aider as a CLI tool.

Keep this info about the user's system in mind:
{{.Platform}}`,
	}

	singleWholeFileFunctionPrompts := &SingleWholeFileFunctionPrompts{
		BasePrompts: *basePrompts,
		MainSystem: `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'write_file' function to update the file to make the changes.`,
		SystemReminder: `ONLY return code using the 'write_file' function.
NEVER return code outside the 'write_file' function.`,
		RedactedEditMessage: "No changes are needed.",
	}

	editBlockFencedPrompts := &EditBlockFencedPrompts{
		EditBlockPrompts: *editBlockPrompts,
	}

	return &PromptsConfig{
		BasePrompts:                    basePrompts,
		ArchitectPrompts:               architectPrompts,
		AskPrompts:                     askPrompts,
		EditBlockPrompts:               editBlockPrompts,
		EditBlockFencedPrompts:         editBlockFencedPrompts,
		EditBlockFunctionPrompts:       editBlockFuncPrompts,
		EditorEditBlockPrompts:         editorEditBlockPrompts,
		EditorWholeFilePrompts:         editorWholeFilePrompts,
		HelpPrompts:                    helpPrompts,
		SingleWholeFileFunctionPrompts: singleWholeFileFunctionPrompts,
		WholeFilePrompts:               wholeFilePrompts,
	}
}
