package parser

import (
	"reflect"
	"testing"

	"github.com/y0ug/ai-helper/internal/coder/diff"
)

func TestParser_ParseResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     []diff.Section
	}{
		{
			name: "single file change",
			response: `test.go
<source>go
<<<<<<< SEARCH
old code
=======
new code
>>>>>>> REPLACE
</source>`,
			want: []diff.Section{
				{
					Filename:     "test.go",
					Language:     "go",
					SearchBlock:  "old code",
					ReplaceBlock: "new code",
				},
			},
		},
		{
			name: "multiple file changes",
			response: `main.go
<source>go
<<<<<<< SEARCH
func old() {}
=======
func new() {}
>>>>>>> REPLACE
</source>
test.go
<source>go
<<<<<<< SEARCH
test old
=======
test new
>>>>>>> REPLACE
</source>`,
			want: []diff.Section{
				{
					Filename:     "main.go",
					Language:     "go",
					SearchBlock:  "func old() {}",
					ReplaceBlock: "func new() {}",
				},
				{
					Filename:     "test.go",
					Language:     "go",
					SearchBlock:  "test old",
					ReplaceBlock: "test new",
				},
			},
		},
		{
			name:     "empty response",
			response: "",
			want:     nil,
		},
		{
			name: "backtick syntax - single file",
			response: `test.go
 ` + "```" + `go
 <<<<<<< SEARCH
 old code
 =======
 new code
 >>>>>>> REPLACE
 ` + "```" + ``,
			want: []diff.Section{
				{
					Filename:     "test.go",
					Language:     "go",
					SearchBlock:  "old code",
					ReplaceBlock: "new code",
				},
			},
		},
		{
			name: "mixed syntax - source and backticks",
			response: `main.go
 <source>go
 <<<<<<< SEARCH
 func old() {}
 =======
 func new() {}
 >>>>>>> REPLACE
 </source>
 test.go
 ` + "```" + `go
 <<<<<<< SEARCH
 test old
 =======
 test new
 >>>>>>> REPLACE
 ` + "```" + ``,
			want: []diff.Section{
				{
					Filename:     "main.go",
					Language:     "go",
					SearchBlock:  "func old() {}",
					ReplaceBlock: "func new() {}",
				},
				{
					Filename:     "test.go",
					Language:     "go",
					SearchBlock:  "test old",
					ReplaceBlock: "test new",
				},
			},
		},
		{
			name:     "empty response",
			response: "",
			want:     nil,
		},
		{
			name: "invalid format - missing markers",
			response: `test.go
<source>go
some code without markers
</source>`,
			want: nil,
		},
		{
			name: "multiple languages",
			response: `main.go
<source>go
<<<<<<< SEARCH
old go
=======
new go
>>>>>>> REPLACE
</source>
script.py
<source>python
<<<<<<< SEARCH
old python
=======
new python
>>>>>>> REPLACE
</source>`,
			want: []diff.Section{
				{
					Filename:     "main.go",
					Language:     "go",
					SearchBlock:  "old go",
					ReplaceBlock: "new go",
				},
				{
					Filename:     "script.py",
					Language:     "python",
					SearchBlock:  "old python",
					ReplaceBlock: "new python",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			got := p.ParseResponse(tt.response)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_parseSection(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		language string
		content  string
		want     *diff.Section
	}{
		{
			name:     "valid section",
			filename: "test.go",
			language: "go",
			content: `<<<<<<< SEARCH
old code
=======
new code
>>>>>>> REPLACE`,
			want: &diff.Section{
				Filename:     "test.go",
				Language:     "go",
				SearchBlock:  "old code",
				ReplaceBlock: "new code",
			},
		},
		{
			name:     "missing search marker",
			filename: "test.go",
			language: "go",
			content: `some code
=======
new code
>>>>>>> REPLACE`,
			want: nil,
		},
		{
			name:     "missing replace marker",
			filename: "test.go",
			language: "go",
			content: `<<<<<<< SEARCH
old code
=======
some code`,
			want: nil,
		},
		{
			name:     "empty content",
			filename: "test.go",
			language: "go",
			content:  "",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			got := p.parseSection(tt.filename, tt.language, tt.content)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSection() = %v, want %v", got, tt.want)
			}
		})
	}
}
