package coder

import "github.com/y0ug/ai-helper/internal/coder/prompts"

type EditBlockCoder struct {
	BaseCoder
	prompts *prompts.EditBlockPrompts
}

func NewEditBlockCoder(opts CoderOptions) *EditBlockCoder {
	return &EditBlockCoder{
		BaseCoder: *NewBaseCoder(opts),
		prompts:   prompts.NewEditBlockPrompts(),
	}
}

func (c *EditBlockCoder) GetEdits(content string) ([]Edit, error) {
	// Parse content for SEARCH/REPLACE blocks using regex
	// Return slice of Edit structs
	return parseSearchReplaceBlocks(content)
}

func (c *EditBlockCoder) ApplyEdits(edits []Edit) error {
	for _, edit := range edits {
		if err := c.applyEdit(edit); err != nil {
			return err
		}
	}
	return nil
}

func (c *EditBlockCoder) applyEdit(edit Edit) error {
	// Apply the edit to the file
	// Handle file creation if needed
	// Update git repo if needed
	return nil
}
