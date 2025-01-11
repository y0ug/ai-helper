package diff

import (
    "testing"
    "reflect"
)

func TestGenerator_ApplyChanges(t *testing.T) {
    tests := []struct {
        name     string
        files    map[string]string
        sections []Section
        want     map[string]string
        wantErr  bool
    }{
        {
            name: "simple replacement",
            files: map[string]string{
                "test.py": "def hello():\n    print('hello')\n",
            },
            sections: []Section{
                {
                    Filename:     "test.py",
                    SearchBlock:  "print('hello')",
                    ReplaceBlock: "print('world')",
                },
            },
            want: map[string]string{
                "test.py": "def hello():\n    print('world')\n",
            },
            wantErr: false,
        },
        {
            name: "file not found",
            files: map[string]string{
                "test.py": "content",
            },
            sections: []Section{
                {
                    Filename:     "missing.py",
                    SearchBlock:  "old",
                    ReplaceBlock: "new",
                },
            },
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            g := NewGenerator()
            got, err := g.ApplyChanges(tt.files, tt.sections)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ApplyChanges() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ApplyChanges() = %v, want %v", got, tt.want)
            }
        })
    }
}
