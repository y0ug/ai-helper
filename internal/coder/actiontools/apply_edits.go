package actiontools

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/responseextractor"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

type ActionTools struct {
	logger *slog.Logger
}

func NewActionTools(logger *slog.Logger) *ActionTools {
	return &ActionTools{
		logger: logger,
	}
}

func (c *ActionTools) ApplyEdits(
	fm filemanager.FileManager,
	dryrun bool,
	edits ...responseextractor.Edit,
) ([]responseextractor.ParsedAction, error) {
	actions := []responseextractor.ParsedAction{}
	hasError := false
	for _, edit := range edits {
		content, isEditable, err := fm.Get(edit.Filename)
		if err != nil {
			c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
			c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
			content = ""
			fm.Add(edit.Filename, false)
		} else if !isEditable {
			c.logger.Info("file is read-only we will not edit it", "filename", edit.Filename)
			err := fmt.Errorf("file %s is read-only we will not edit it", edit.Filename)
			actions = append(
				actions,
				*responseextractor.NewActionError(err),
			)
			hasError = true
			continue
		}

		newContent := responseextractor.ApplyEdit(string(content), edit.Original, edit.Updated)
		if newContent == string(content) {
			err = fmt.Errorf("new content is the same as the original content")
			actions = append(actions, *responseextractor.NewActionError(err))
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
					actions = append(
						actions,
						*responseextractor.NewActionError(fmt.Errorf("failed to write file %s %+v", edit.Filename, err)),
					)
					hasError = true
				}
			}
		}
	}
	if hasError {
		return actions, fmt.Errorf("failed to apply some edits check result actions")
	}
	return actions, nil
}
