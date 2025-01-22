package coder

import "github.com/y0ug/ai-helper/internal/coder/prompts"

type ArchitectCoder struct {
	BaseCoder
	prompts *prompts.ArchitectPrompts
}

func NewArchitectCoder(opts CoderOptions) *ArchitectCoder {
	return &ArchitectCoder{
		BaseCoder: *NewBaseCoder(opts),
		prompts:   prompts.NewArchitectPrompts(),
	}
}

func (c *ArchitectCoder) GetEdits(content string) ([]Edit, error) {
	// For architect, we don't directly parse edits
	// Instead, we create a new editor coder to handle the changes
	// if !c.io.ConfirmAsk("Edit the files?") {
	// 	return nil, nil
	// }

	editorOpts := CoderOptions{
		MainModel:  c.mainModel.EditorModel,
		EditFormat: c.mainModel.EditorEditFormat,
		IO:         c.io,
		// ... other options
	}

	editor, err := NewCoder(editorOpts)
	if err != nil {
		return nil, err
	}

	return editor.GetEdits(content)
}
