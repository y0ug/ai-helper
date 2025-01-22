package prompts

import (
	"bytes"
	"log/slog"
	"text/template"

	"github.com/y0ug/ai-helper/internal/coder/models"
)

type TemplateHandler struct {
	prompts   *EditBlockPrompts
	mainModel *models.Model
	data      TemplateData
	logger    *slog.Logger
}

func NewTemplateHandler(
	prompts *EditBlockPrompts,
	model *models.Model,
	initialData TemplateData,
	logger *slog.Logger,
) *TemplateHandler {
	th := &TemplateHandler{
		mainModel: model,
		prompts:   prompts,
		data:      initialData,
		logger:    logger,
	}
	for key, value := range initialData {
		th.Set(key, value)
	}
	// TODO: check that language and platform are set
	th.UpdateData()
	return th
}

type TemplateVar string

const (
	VarLanguage       TemplateVar = "Language"
	VarPlatform       TemplateVar = "Platform"
	VarFence0         TemplateVar = "Fence0"
	VarFence1         TemplateVar = "Fence1"
	VarLazyPrompt     TemplateVar = "LazyPrompt"
	VarShellCmdPrompt TemplateVar = "ShellCmdPrompt"
	// Add other vars as needed
)

func (c *TemplateHandler) UpdateData() {
	c.Set("LazyPrompt", c.getLazyPrompt())
	c.Set("ShellCmdPrompt", c.Render(c.prompts.ShellCmdPrompt))
	c.Set("ShellCmdReminder", c.Render(c.prompts.ShellCmdReminder))
}

func (c *TemplateHandler) SetFence(fence [2]string) {
	c.Set("Fence0", fence[0])
	c.Set("Fence1", fence[1])
}

func (c *TemplateHandler) getLazyPrompt() string {
	if c.mainModel.Lazy {
		return c.Render(c.prompts.LazyPrompt)
	}
	return ""
}

func (th *TemplateHandler) Set(key string, value string) {
	th.data[string(key)] = value
}

func (th *TemplateHandler) RenderData(templateText string, data map[string]string) string {
	for key, value := range th.data {
		if _, ok := data[key]; !ok {
			data[key] = value
		}
	}
	tmpl, err := template.New("prompt").Parse(templateText)
	if err != nil {
		th.logger.Error("Failed to parse template", "error", err)
		return templateText
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		th.logger.Error("Failed to render template", "error", err)
		return templateText
	}

	return buf.String()
}

func (th *TemplateHandler) Render(templateText string) string {
	return th.RenderData(templateText, make(map[string]string))
}

// TemplateData type changed to map for flexibility
type TemplateData map[string]string
