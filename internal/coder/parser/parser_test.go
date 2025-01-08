package parser

import (
    "testing"
    "reflect"
    "github.com/yourusername/yourproject/internal/coder/diff"
)

func TestParser_ParseResponse(t *testing.T) {
    tests := []struct {
        name     string
        response string
        want     []diff.Section
    }{
        {
            name: "simple replacement",
            response: `test.py
` + "```" + `python
<<<<<<< SEARCH
print('hello')
