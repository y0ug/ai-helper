package context

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/y0ug/ai-helper/internal/config"
)

type VariableProcessor struct {
	command *config.Command
}

func NewVariableProcessor(cmd *config.Command) *VariableProcessor {
	return &VariableProcessor{
		command: cmd,
	}
}

// Process processes variables and returns a map of variable values
func (vp *VariableProcessor) Process(
	templateID string, args map[string]interface{},
) (map[string]interface{}, error) {
	variables, err := vp.getVariablesFromCommand(templateID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]interface{})

	for _, v := range variables {
		value, err := vp.processVariable(v, args)
		if err != nil {
			return nil, err
		}
		results[v.Name] = value
	}

	return results, nil
}

func (vp *VariableProcessor) getVariablesFromCommand(templateID string) ([]config.Variable, error) {
	if template, ok := vp.command.Templates[templateID]; ok {
		return template.Variables, nil
	}
	return nil, fmt.Errorf("template %s not found in command", templateID)
}

func (vp *VariableProcessor) processVariable(
	v config.Variable,
	args map[string]interface{},
) (interface{}, error) {
	types := v.GetTypes()
	if len(types) == 0 {
		return "", nil
	}

	for _, t := range types {
		value, ok := vp.tryGetValue(t, v, args)
		if ok {
			return value, nil
		}
	}

	return "", fmt.Errorf(
		"no value found for variable %s after trying types: %v",
		v.Name,
		types,
	)
}

func (vp *VariableProcessor) tryGetValue(
	t string,
	v config.Variable,
	args map[string]interface{},
) (interface{}, bool) {
	switch t {
	case config.VarTypeExec:
		if v.Exec != "" {
			cmd := exec.Command("sh", "-c", v.Exec)
			if output, err := cmd.Output(); err == nil {
				return strings.TrimSpace(string(output)), true
			}
		}
	case config.VarTypeArg:
		if val, ok := args[v.Name]; ok {
			return val, true
		}
	case config.VarTypeStdin:
		if val, err := readStdin(); err == nil {
			return val, true
		}
	}
	return "", false
}

func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", fmt.Errorf("failed to read stdin: %w", err)
			}
			builder.WriteString(line)
		}

		if builder.Len() > 0 {
			return strings.TrimSpace(builder.String()), nil
		}
	}
	return "", fmt.Errorf("no input found")
}
