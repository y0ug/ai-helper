# Project Technical Specification

## Core Components

### 1. AI Package (internal/ai/)

#### Client
```go
type Client struct {
    provider Provider
    model    *Model
    stats    *stats.Tracker
    agents   map[string]*Agent // Track active agents by ID
}
```

#### Agent
```go
type Agent struct {
    ID                string               // Unique identifier for this agent/session
    Model             *Model               // The AI model being used
    Messages          []Message            // Conversation history
    Command           *config.Command      // Current active command
    TemplateData      *prompt.TemplateData // Data for template processing
    CreatedAt         time.Time            // When the agent was created
    UpdatedAt         time.Time            // Last time the agent was updated
    TotalInputTokens  int                  // Total tokens used in inputs
    TotalOutputTokens int                  // Total tokens used in outputs
}

type AgentState struct {
    ID                string               `json:"id"`
    ModelName         string               `json:"model"`
    Messages          []Message            `json:"messages"`
    Command           *config.Command      `json:"command,omitempty"`
    TemplateData      *prompt.TemplateData `json:"-"`
    CreatedAt         time.Time            `json:"created_at"`
    UpdatedAt         time.Time            `json:"updated_at"`
    TotalInputTokens  int                  `json:"total_input_tokens"`
    TotalOutputTokens int                  `json:"total_output_tokens"`
}
```

#### Model
```go
type Model struct {
    Provider string
    Name     string
    info     *Info
}

type Info struct {
    MaxTokens                 int
    MaxInputTokens            int
    MaxOutputTokens           int
    InputCostPerToken         float64
    OutputCostPerToken        float64
    LiteLLMProvider           string
    Mode                      string
    SupportsFunctionCalling   bool
    SupportsVision            bool
}
```

#### Message/Request/Response Types
```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type Request struct {
    Model    string    `json:"model"`
    System   string    `json:"system,omitempty"`
    Messages []Message `json:"messages"`
}

type Response struct {
    Content      string
    InputTokens  int
    OutputTokens int
    CachedTokens int
    Cost         *float64
    Error        error
}
```

### 2. Config Package (internal/config/)

#### Config Types
```go
type Config struct {
    Commands map[string]Command `yaml:"commands" json:"commands"`
}

type Command struct {
    Description  string     `yaml:"description,omitempty"   json:"description,omitempty"`
    System       string     `yaml:"system,omitempty"        json:"system,omitempty"`
    Prompt       string     `yaml:"prompt"                  json:"prompt"`
    Variables    []Variable `yaml:"variables,omitempty"     json:"variables,omitempty"`
    Input        bool       `yaml:"input,omitempty"         json:"input,omitempty"`
    InputCommand string     `yaml:"input_command,omitempty" json:"input_command,omitempty"`
    Files        []string   `yaml:"files,omitempty"         json:"files,omitempty"`
}

type Variable struct {
    Name string `yaml:"name,omitempty" json:"name,omitempty"`
    Type string `yaml:"type,omitempty" json:"type,omitempty"`
    Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`
}
```

### 3. Stats Package (internal/stats/)

#### Statistics Types
```go
type Stats struct {
    Providers map[string]*ProviderStats `json:"providers"`
    mu        sync.RWMutex             `json:"-"`
}

type ProviderStats struct {
    Queries       int64                    `json:"queries"`
    InputTokens   int64                    `json:"input_tokens"`
    OutputTokens  int64                    `json:"output_tokens"`
    Cost          float64                  `json:"cost"`
    LastUsed      time.Time                `json:"last_used"`
    Commands      map[string]*CommandStats `json:"commands"`
}

type CommandStats struct {
    Count        int64     `json:"count"`
    InputTokens  int64     `json:"input_tokens"`
    OutputTokens int64     `json:"output_tokens"`
    Cost         float64   `json:"cost"`
    LastUsed     time.Time `json:"last_used"`
}
```

### 4. Prompt Package (internal/prompt/)

#### Template Types
```go
type TemplateData struct {
    Input string
    Env   map[string]string
    Files map[string]string
    Vars  map[string]interface{}
}
```

## Key Relationships

1. Client manages multiple Agents
2. Each Agent has one Model
3. Models interact with Providers
4. Commands are defined in Config
5. Stats track usage per Provider and Command
6. TemplateData is used by both Agent and Prompt processing

## Extension Points

1. Provider Interface - Add new AI providers
2. Command Configuration - Define new command types
3. Template Functions - Add new template capabilities
4. Stats Tracking - Extend monitoring metrics

## Data Flow

1. Client receives command request
2. Agent processes command using Model
3. Provider generates response
4. Stats track usage
5. Response returned to client

This specification provides a complete view of the system's structure and capabilities for LLM-based feature design.
