# AI Helper

A powerful command-line tool for seamless interaction with AI providers (OpenAI, Anthropic, OpenRouter) through a configurable interface.

## Features

- 🤖 Multiple AI Provider Support
  - OpenAI (GPT-3.5, GPT-4)
  - Anthropic (Claude)
  - OpenRouter (unified access to multiple models)
- ⚙️ Rich Configuration
  - YAML and JSON support
  - Custom command definitions
  - Variable substitution
  - Command execution in prompts
- 📊 Usage Statistics
  - Token counting
  - Cost tracking
  - Command usage history
- 🛠️ Developer Tools
  - Git commit message generation
  - Code explanation
  - Documentation help

## Quick Start

### Installation

```bash
go install github.com/y0ug/ai-helper/cmd/ai-helper@latest
```

### Basic Usage

```bash
# Get help
ai-helper --help

# List available commands
ai-helper --list

# Ask a question
ai-helper ask "What is the capital of France?"

# View usage statistics
ai-helper --stats
```

## Configuration

The tool searches for configuration in:
1. Current directory (`ai-helper.yaml` or `ai-helper.json`)
2. `$XDG_CONFIG_HOME/ai-helper/` or `~/.config/ai-helper/`

### Example Configuration

```yaml
commands:
  ask:
    variables:
      - name: Input
        type: stdin|arg
    description: Ask a computer expert
    prompt: |
      I want you to act as a computer expert. 
      I will ask you questions about Linux/Windows shell commands, scripting, 
      and system administration tasks, vim. For each question:

      1. If it's a command request:
        - Provide the exact command without explanation
        - Add only critical flags/options
        - Include a single-line comment explaining the command (if needed)
        - Format as a code block

      2. If it's a scripting question:
        - Give the shortest working solution
        - Include only essential error handling
        - Format as a code block

      3. If it's a conceptual question:
        - Provide a max 2-sentence answer
        - Include a command example if relevant

      Don't explain basic concepts unless asked. Don't provide alternatives unless requested. Focus on the most direct solution.

      {{ .Input }}

  summarize:
    description: Summarize a text
    variables:
      - name: Input
        type: stdin|arg
    prompt: |
      Please summarize the following text:
      {{ .Input }}

  git-commit:
    variables:
      - name: RecentCommits
        type: exec
        exec: git log -3 --pretty=format:"%s"
      - name: Input
        type: stdin|arg|exec
        exec: git diff --cached
    prompt: |
      Generate a git commit message following this structure:

      1. First line: conventional commit format (type: concise description) (remember to use semantic
      types like feat, fix, docs, style, refactor, perf, test, chore, etc.)
      2. Optional bullet points if more context helps:
        - Keep the second line blank
        - Keep them short and direct
        - Focus on what changed
        - Always be terse
        - Don't overly explain
        - Drop any fluffy or formal language

      **Important:** Before generating the commit message, analyze the git diff for sensitive information.
      If the diff contains any secrets (e.g., API keys, passwords, tokens, private keys, or anything resembling credentials),
      return the following exact response and nothing else: 

      "#ERROR# 🚨 Potential secret detected in the git diff. Commit aborted!"

      Return **ONLY** the commit message if no secret are detected - no introduction, no explanation, no quotes around it.

      **Patterns to watch for include (but are not limited to):**
      - Any variable containing words like `token`, `password`, `secret`, `apikey`, `private_key`
      - Any base64-encoded or hex-encoded long strings
      - Any obvious credentials in JSON, YAML, ENV files, or config changes

      If no secrets are detected, proceed with generating the commit message.

      Recent commits from this repo (for style reference):
      {{ .RecentCommits }}

      Here's the diff:

      {{ .Input }}
```

## Advanced Usage

```bash
# Pipe input
echo "Hello world" | ai-helper translate

# Git integration
git diff | ai-helper git-commit

# Save output to file
ai-helper ask "Write a poem" --output poem.txt

# Show token usage and cost
ai-helper -v ask "What is Docker?"
```

## Environment Setup

Required environment variables:

```bash
# Choose your AI provider/model
export AI_MODEL="openai/gpt-3.5-turbo"  # or "anthropic/claude-2", "openrouter/anthropic/claude-2"

# Set API keys for your chosen provider
export OPENAI_API_KEY="your-key"        # For OpenAI
export ANTHROPIC_API_KEY="your-key"     # For Anthropic
export OPENROUTER_API_KEY="your-key"    # For OpenRouter
```

## Shell Completion

Generate shell completion scripts:

```bash
# For zsh
ai-helper --completion zsh > ~/.zsh/completion/_ai-helper

# Direct shell sourcing
source <(ai-helper -completion zsh)

# Using zinit
zinit has"ai-helper" id-as"ai-helper-completion" \
    atclone"ai-helper -completion zsh > _ai_helper" \
    src"_ai_helper" \
    @zdharma-continuum/null
```

## Shell Integration

Add these aliases to your shell configuration:

```bash
# Git commit with AI-generated message
alias gca='MSG=$(ai-helper git-commit "$(git diff --cached)") && [[ -n "$MSG" && "$MSG" != "#ERROR#"* ]] && git commit -m "$MSG" || echo -e "ERROR\n$MSG"'
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a Pull Request

## License

[MIT License](LICENSE)
