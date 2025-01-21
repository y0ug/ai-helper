package prompts

func NewEditBlockPrompts() *EditBlockPrompts {
	prompts := &EditBlockPrompts{
		BasePrompts: *NewBasePrompts(),
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
	prompts.ExampleMessages = getEditBlockExampleMessages()
	return prompts
}

func getEditBlockExampleMessages() []Message {
	return []Message{
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
}
