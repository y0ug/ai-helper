package editblock

import (
	"math"
	"strings"
)

func isShellBlockStart(line string) bool {
	shellPrefixes := []string{"```bash", "```sh", "```shell", "```cmd", "```batch"}
	for _, prefix := range shellPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func extractShellCommand(lines []string, i int) (string, int) {
	var cmdLines []string
	i++ // Skip opening fence
	for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
		cmdLines = append(cmdLines, lines[i])
		i++
	}
	i++ // Skip closing fence
	return strings.Join(cmdLines, "\n"), i
}

func DoReplace(content, original, updated string) string {
	original = stripQuotedWrapping(original)
	updated = stripQuotedWrapping(updated)

	// First try exact match
	if idx := strings.Index(content, original); idx != -1 {
		return content[:idx] + updated + content[idx+len(original):]
	}

	// Try flexible whitespace matching
	return replaceWithFlexibleWhitespace(content, original, updated)
}

func stripQuotedWrapping(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Remove fences
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func replaceWithFlexibleWhitespace(content, original, updated string) string {
	contentLines := splitLines(content)
	origLines := splitLines(original)
	updatedLines := splitLines(updated)

	// Find best matching block with flexible whitespace
	bestScore := math.MaxFloat64
	bestIndex := -1

	for i := 0; i < len(contentLines)-len(origLines); i++ {
		score := compareBlocks(contentLines[i:i+len(origLines)], origLines)
		if score < bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	if bestIndex == -1 {
		return content
	}

	var result []string
	result = append(result, contentLines[:bestIndex]...)
	result = append(result, updatedLines...)
	result = append(result, contentLines[bestIndex+len(origLines):]...)
	return strings.Join(result, "\n")
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func compareBlocks(a, b []string) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var total float64
	for i := range a {
		total += lineDistance(a[i], b[i])
	}
	return total
}

func lineDistance(a, b string) float64 {
	aClean := strings.TrimSpace(a)
	bClean := strings.TrimSpace(b)
	if aClean == bClean {
		return 0.1 * float64(len(a)-len(aClean))
	}
	return 1.0
}
