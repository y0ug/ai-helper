# AI Helper

A powerful command-line tool for seamless interaction with AI providers (OpenAI, Anthropic, OpenRouter) through a configurable interface.

## Features

- 🤖 Multiple AI Provider Support
  - OpenAI (GPT-3.5, GPT-4)
  - Anthropic (Claude)
  - Google (Gemini)
  - DeepSeek (Chat)
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

#### Using Go

```bash
go install github.com/y0ug/ai-helper/cmd/ai-helper@latest
```

#### Using Install Script (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/y0ug/ai-helper/main/install.sh | bash
```

This will install the binary to `~/.local/bin`. Make sure this directory is in your PATH.

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
    input: true
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

  whoami:
    description: Demo whoami
    variables:
      - name: WhoAmi
        type: exec
        exec: whoami
    prompt: |
      **Return ONLY** with my name on json format.
      I'm {{ .WhoAmi }}. Can you say who I'm?

  analyze:
    description: "Analyze code files"
    prompt: |
      Please analyze these code files:

      {{range $filePath, $content := .Files}}
      File: {{$filePath}}
      ```
      {{$content}}
      ```

      {{end}}
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

# Analyze multiple files
ai-helper analyze file1.go file2.go file3.go

# Get system information
ai-helper whoami
```

## Environment Setup

Required environment variables:

```bash
# Choose your AI provider/model
export AI_MODEL="openai/gpt-3.5-turbo"  # or "openai/gpt-4", "anthropic/claude-3-sonnet-20241022", 
                                       # "google/gemini-pro", "google/gemini-exp-1206",
                                       # "deepseek/chat", "openrouter/anthropic/claude-2"

# Set API keys for your chosen provider
export OPENAI_API_KEY="your-key"        # For OpenAI
export ANTHROPIC_API_KEY="your-key"     # For Anthropic
export GOOGLE_API_KEY="your-key"        # For Google Gemini
export OPENROUTER_API_KEY="your-key"    # For OpenRouter
export DEEPSEEK_API_KEY="your-key"      # For DeepSeek
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
