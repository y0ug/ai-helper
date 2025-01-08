package coder

import (
    "context"
    "fmt"
    "github.com/yourusername/yourproject/internal/ai"
    "github.com/yourusername/yourproject/internal/coder/diff"
    "github.com/yourusername/yourproject/internal/coder/parser"
    "github.com/yourusername/yourproject/internal/coder/prompts"
    "github.com/yourusername/yourproject/internal/prompt"
)

type Service struct {
    parser  *parser.Parser
    diff    *diff.Generator
    prompts *prompts.Manager
}

type Response struct {
    Analysis      string
    Changes       string
    ModifiedFiles map[string]string
    Patches       map[string]string
}

func NewService() *Service {
    return &Service{
        parser:  parser.New(),
        diff:    diff.NewGenerator(),
        prompts: prompts.NewManager(),
    }
}

func (s *Service) ProcessRequest(ctx context.Context, agent *ai.Agent, request string, files map[string]string) (*Response, error) {
    if err := s.prompts.LoadTemplates(); err != nil {
        return nil, fmt.Errorf("failed to load templates: %w", err)
    }

    data := &prompt.TemplateData{
        Files: files,
        Vars: map[string]interface{}{
            "Request": request,
        },
    }

    // Get analysis
    analysisPrompt, err := s.prompts.Execute("analyze", data)
    if err != nil {
        return nil, fmt.Errorf("failed to generate analysis prompt: %w", err)
    }

    analysis, err := s.executePrompt(ctx, agent, analysisPrompt)
    if err != nil {
        return nil, fmt.Errorf("failed to get analysis: %w", err)
    }

    // Generate changes
    data.Vars["Analysis"] = analysis
    changePrompt, err := s.prompts.Execute("generate", data)
    if err != nil {
        return nil, fmt.Errorf("failed to generate change prompt: %w", err)
    }

    changes, err := s.executePrompt(ctx, agent, changePrompt)
    if err != nil {
        return nil, fmt.Errorf("failed to generate changes: %w", err)
    }

    // Parse and apply changes
    sections := s.parser.ParseResponse(changes)
    modifiedFiles, err := s.diff.ApplyChanges(files, sections)
    if err != nil {
        return nil, fmt.Errorf("failed to apply changes: %w", err)
    }

    // Generate patches
    patches := make(map[string]string)
    for filename, modifiedContent := range modifiedFiles {
        if original, exists := files[filename]; exists && original != modifiedContent {
            patches[filename] = s.diff.GeneratePatch(original, modifiedContent)
        }
    }

    return &Response{
        Analysis:      analysis,
        Changes:       changes,
        ModifiedFiles: modifiedFiles,
        Patches:       patches,
    }, nil
}

func (s *Service) executePrompt(ctx context.Context, agent *ai.Agent, prompt string) (string, error) {
    msg := ai.Message{
        Role:    "user",
        Content: prompt,
    }
    
    agent.Messages = append(agent.Messages, msg)
    resp, err := agent.Model.SendMessages(agent.Messages)
    if err != nil {
        return "", err
    }
    
    agent.Messages = append(agent.Messages, ai.Message{
        Role:    "assistant",
        Content: resp.Content,
    })
    
    return resp.Content, nil
}
