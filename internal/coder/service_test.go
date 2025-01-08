package coder

import (
	"context"
	"testing"

	"github.com/y0ug/ai-helper/internal/ai"
	"go.uber.org/mock/gomock"
)

func TestService_ProcessRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
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
			// Create mock client
			mockClient := ai.NewMockAIClient(ctrl)
			
			// Setup expected calls
			for _, resp := range tt.mockResp {
				mockClient.EXPECT().
					GenerateWithMessages(gomock.Any(), gomock.Any()).
					Return(ai.Response{Content: resp}, nil)
			}

			// Create mock agent
			agent := &ai.Agent{
				Client:   mockClient,
				Messages: []ai.Message{},
			}

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
