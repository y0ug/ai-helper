package executors

import (
	"fmt"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/extractor"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

func (c *Executor) ApplyEdits(
	fm filemanager.FileManager,
	dryrun bool,
	edits ...actions.ApplyEdit,
) ([]actions.Action[any], error) {
	results := make([]actions.Action[any], 0)

	hasError := false
	for _, edit := range edits {
		content, isEditable, err := fm.Get(edit.Filename)
		if err != nil {
			c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
			c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
			content = ""
			fm.Add(edit.Filename, false)
		} else if !isEditable {
			err := fmt.Errorf("file %s is read-only we will not edit it", edit.Filename)
			c.logger.Error("file is read-only", "error", err)
			// actions = append(
			// 	actions,
			// 	*responseextractor.NewActionError(err),
			// )
			hasError = true
			continue
		}

		newContent := extractor.ApplyEdit(string(content), edit.Original, edit.Updated)
		if newContent == string(content) {
			err = fmt.Errorf("new content is the same as the original content")
			c.logger.Error("new content is the same as the original content", "error", err)
			continue
			// actions = append(actions, *responseextractor.NewActionError(err))
		}

		c.logger.Info(
			"applying edit to file",
			"filename",
			edit.Filename,
			"content",
			string(content),
			"new_content",
			newContent,
			"original",
			edit.Original,
			"updated",
			edit.Updated,
		)
		if newContent != string(content) {
			if !dryrun {
				err := fm.Write(edit.Filename, newContent)
				if err != nil {
					// actions = append(
					// 	actions,
					// 	*responseextractor.NewActionError(fmt.Errorf("failed to write file %s %+v", edit.Filename, err)),
					// )
					err = fmt.Errorf("failed to write file %s %+v", edit.Filename, err)
					c.logger.Error("failed to write file", "error", err)
					hasError = true
					continue
				}
			}
		}
	}
	if hasError {
		return results, fmt.Errorf("failed to apply some edits check result actions")
	}
	return results, nil
}
