package main

func generateZshCompletion() string {
	return `#compdef ai-helper
compdef _ai_helper ai-helper

_ai_helper() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -s -C \
        "-h[Show help information]" \
        "--help[Show help information]" \
        "-v[Show verbose cost information]" \
        "--verbose[Show verbose cost information]" \
        "-o[Specify output file]:output file:_files" \
        "--output[Specify output file]:output file:_files" \
        "-c[Specify config file]:config file:_files" \
        "--config[Specify config file]:config file:_files" \
        "--completion[Generate shell completion script]:shell:(zsh)" \
        "--list[List available commands]" \
        "--stats[Show usage statistics]" \
        '1: :->cmds' \
        '*::arg:->args'

    case $state in
        cmds)
            local -a commands
            commands=(${(f)"$(ai-helper --list | sed '1d' | sed 's/^  \([^ ]*\)[ ]*/\1:/')"})
            _describe -t commands "ai-helper command" commands
            ;;
        args)
            case $words[1] in
                *)
                    _normal
                    ;;
            esac
            ;;
    esac
}

# don't run the completion function when being source-ed or eval-ed
if [ "$funcstack[1]" = "_ai_helper" ]; then
  _ai_helper "$@" 
fi
`
}
