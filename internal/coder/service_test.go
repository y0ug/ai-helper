package coder

import (
	"context"
	"testing"

	"github.com/y0ug/ai-helper/internal/ai"
)

type mockClient struct {
	responses []string
	current   int
}

func (m *mockClient) GenerateWithMessages(
	messages []ai.Message,
	command string,
) (*ai.Response, error) {
	resp := m.responses[m.current]
	m.current++
	return &ai.Response{Content: resp}, nil
}

func TestService_ProcessRequest(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		request  string
		mockResp []string
		want     *Response
		wantErr  bool
	}{
		{
			name: "successful code change",
			files: map[string]string{
				"test.go": "old code",
			},
			request: "update the code",
			mockResp: []string{
				"Analysis of the code",
				`test.go
<source>go
<<<<<<< SEARCH
old code
=======
new code
>>>>>>> REPLACE
</source>`,
			},
			want: &Response{
				Analysis: "Analysis of the code",
				Changes: `test.go
<source>go
<<<<<<< SEARCH
old code
=======
new code
>>>>>>> REPLACE
</source>`,
				ModifiedFiles: map[string]string{
					"test.go": "new code",
				},
				Patches: map[string]string{
					"test.go": "--- Original\n+++ Modified\nnew code",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService()
			got, err := s.ProcessRequest(context.Background(), agent, tt.request, tt.files)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil {
				if got.Analysis != tt.want.Analysis {
					t.Errorf("Analysis = %v, want %v", got.Analysis, tt.want.Analysis)
				}
				if got.Changes != tt.want.Changes {
					t.Errorf("Changes = %v, want %v", got.Changes, tt.want.Changes)
				}
			}
		})
	}
}
