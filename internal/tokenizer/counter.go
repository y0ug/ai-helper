package tokenizer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/pkoukk/tiktoken-go"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

func GetEncoding(modelName string) string {
	if encodingName, ok := tiktoken.MODEL_TO_ENCODING[modelName]; ok {
		return encodingName
	} else {
		for prefix, encodingName := range tiktoken.MODEL_PREFIX_TO_ENCODING {
			if strings.HasPrefix(modelName, prefix) {
				return encodingName
			}
		}
	}
	// Handle gpt-4o which uses o200k_base encoding
	if strings.HasPrefix(modelName, "gpt-4o") {
		return "o200k_base"
	}
	return ""
}

func CountRequest(
	tkm *tiktoken.Tiktoken,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	countResponseTokens bool,
) (numTokens int) {
	msgsTokens, _, hasSystemPrompt := CountMessage(tkm, false, messages...)
	numTokens += msgsTokens

	toolsTokens := CountTool(tkm, tools...)
	fmt.Println("toolsTokens: ", toolsTokens)
	if len(tools) > 0 {
		toolsTokens += 9 // Additional tokens for function definition of tools
	}
	numTokens += toolsTokens

	// If there's a system message and tools are present, subtract four tokens
	if hasSystemPrompt && len(tools) > 0 {
		numTokens -= 4
	}

	// Add 3 tokens only if not counting response tokens
	if !countResponseTokens {
		numTokens += 3 // every reply is primed with <|start|>assistant<|message|>
	}
	return
}

// ... [Other functions remain the same until formatType]

func formatType(prop *jsonschema.Schema, indent int) string {
	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			enumValues := make([]string, 0, len(prop.Enum))
			for _, v := range prop.Enum {
				strVal, ok := v.(string)
				if !ok {
					fmt.Printf("non-string value in enum: %T\n", v)
					continue
				}
				enumValues = append(enumValues, fmt.Sprintf(`"%s"`, strVal))
			}
			return strings.Join(enumValues, " | ")
		}
		return "string"
	case "array":
		if prop.Items != nil {
			return fmt.Sprintf("%s[]", formatType(prop.Items, indent))
		}
		return "any[]"
	case "object":
		return fmt.Sprintf(
			"{\n%s\n%s}",
			formatObjectParameters(*prop, indent+4),
			strings.Repeat(" ", indent),
		)
	case "integer", "number":
		if len(prop.Enum) > 0 {
			var strVals []string
			for _, val := range prop.Enum {
				switch v := val.(type) {
				case float64:
					strVals = append(strVals, fmt.Sprintf("%d", int64(v)))
				case int64:
					strVals = append(strVals, fmt.Sprintf("%d", v))
				default:
					fmt.Printf("unsupported enum value type: %T\n", val)
				}
			}
			return strings.Join(strVals, " | ")
		}
		return "number"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	default:
		return "any"
	}
}

// Update ContentTypeToolUse handling in CountMessage
func CountMessage(
	tkm *tiktoken.Tiktoken,
	isResponseTokens bool,
	msgs ...*chat.ChatMessage,
) (numTokens int, hasFunctionUse, hasSystemPrompt bool) {
	for _, msg := range msgs {
		numTokens += 3 // per Message

		if msg.Role == "system" {
			hasSystemPrompt = true
		}

		// Counting the tokens for the role
		roleTokens := len(tkm.Encode(msg.Role, nil, nil))
		numTokens += roleTokens
		fmt.Printf("% 4d role (%d): %s\n", numTokens, roleTokens, msg.Role)

		msgTokens := 0

		for _, content := range msg.Content {
			curContentTokens := 0
			switch content.Type {
			case chat.ContentTypeText:
				curContentTokens = len(tkm.Encode(content.Text, nil, nil))
			case chat.ContentTypeToolResult:
				curContentTokens = len(tkm.Encode(string(content.Content), nil, nil))
			case chat.ContentTypeToolUse:
				toolUse := struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      content.Name,
					Arguments: string(content.Input),
				}
				input, _ := json.Marshal(toolUse)
				fmt.Println("input: ", string(input))
				curContentTokens = len(tkm.Encode(string(input), nil, nil))
				// curContentTokens += len(tkm.Encode(string(content.Name), nil, nil))
				// curContentTokens += len(tkm.Encode(string(content.Input), nil, nil))
				hasFunctionUse = true
			default:
				fmt.Println("not counted content.Type: ", content.Type)
			}
			numTokens += curContentTokens
			msgTokens += curContentTokens
			fmt.Printf(
				"% 4d context (%s , %d): %s\n",
				numTokens,
				content.Type,
				curContentTokens,
				content.String(),
			)
		}

		if isResponseTokens {
			// No adjustment needed for response tokens
			return msgTokens, hasFunctionUse, hasSystemPrompt
		}

		// Removed incorrect adjustment for assistant messages
	}
	return
}

func CountTool(tkm *tiktoken.Tiktoken, tools ...chat.Tool) int {
	content := formatFunctionDefinitions(tools...)
	fmt.Println(content)
	return len(tkm.Encode(content, nil, nil))
}

func formatFunctionDefinitions(tools ...chat.Tool) string {
	var lines []string

	if len(tools) == 0 {
		return ""
	}

	lines = append(lines, "namespace functions {")
	lines = append(lines, "")

	for _, tool := range tools {
		switch schema := tool.InputSchema.(type) {
		case *jsonschema.Schema:
			if tool.InputSchema == nil {
				continue
			}

			if tool.Description != nil {
				lines = append(lines, fmt.Sprintf("// %s", *tool.Description))
			}

			lines = append(lines, fmt.Sprintf("type %s = (_: {", tool.Name))
			lines = append(lines, formatObjectParameters(*schema, 4))
			lines = append(lines, "}) => any")
			lines = append(lines, "")
		}
	}
	lines = append(lines, "} // namespace functions")
	return strings.Join(lines, "\n")
}

func formatObjectParameters(schema jsonschema.Schema, indent int) string {
	var lines []string
	for pair := schema.Properties.Newest(); pair != nil; pair = pair.Prev() {
		if pair.Value.Description != "" {
			lines = append(
				lines,
				fmt.Sprintf("%s// %s", strings.Repeat(" ", indent), pair.Value.Description),
			)
		}

		question := "?"
		if contains(schema.Required, pair.Key) {
			question = ""
		}

		lines = append(
			lines,
			fmt.Sprintf(
				"%s%s%s: %s,",
				strings.Repeat(" ", indent),
				pair.Key,
				question,
				formatType(pair.Value, indent),
			),
		)
	}
	return strings.Join(lines, "\n")
}

func SliceToType[T any](slice []any) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		val, ok := v.(T)
		if !ok {
			fmt.Printf("non-string value in slice %T\n", v)
			continue
		}
		result[i] = val
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
