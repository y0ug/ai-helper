package mcpclient

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"golang.org/x/exp/jsonrpc2"
)

// MCPClientInterface defines the interface for MCP client operations
type MCPClientInterface interface {
	// Initialize sends the initialize request to the server and stores the capabilities
	Initialize(ctx context.Context) (*ServerInfo, error)
	
	// Ping sends a ping request to check if the server is alive
	Ping(ctx context.Context) error
	
	// ListTools requests the list of available tools from the server
	ListTools(ctx context.Context, cursor *string) ([]Tool, *string, error)
	
	// ListResources requests the list of available resources from the server
	ListResources(ctx context.Context, cursor *string) ([]Resource, *string, error)
	
	// ReadResource reads a specific resource from the server
	ReadResource(ctx context.Context, uri string) (*[]interface{}, error)
	
	// CallTool executes a specific tool with given parameters
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error)
	
	// Close shuts down the MCP client and server
	Close() error
}

type MCPClient struct {
	conn     *jsonrpc2.Connection
	cancelFn context.CancelFunc

	// Track initialization state
	initialized bool

	// Server capabilities received during initialization
	ServerInfo *ServerInfo
}

func FetchAll[T any](
	ctx context.Context,
	fetch func(ctx context.Context, cursor *string) ([]T, *string, error),
) ([]T, error) {
	var allItems []T
	var cursor *string

	for {
		items, nextCursor, err := fetch(ctx, cursor)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: %w", err)
		}

		allItems = append(allItems, items...)

		if nextCursor == nil {
			break
		}

		cursor = nextCursor
	}

	return allItems, nil
}

func logHandler() jsonrpc2.HandlerFunc {
	return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		log.Print("Request received", "Method",
			req.Method, "Id", req.ID, "params", string(req.Params))
		return nil, jsonrpc2.ErrNotHandled
	}
}

// NewMCPClient creates a new MCP client and starts the language server
func NewMCPClient(serverCmd string, args ...string) (MCPClientInterface, error) {
	cmd := exec.Command(serverCmd, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}
	dialer := &StdioStream{
		reader: stdout,
		writer: stdin,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// HeaderFramer is the jsonrpc2.Framer options
	// That's what MCP servers are expecting
	debug := false
	framer := NewLineRawFramer()
	if debug {
		framer = &LoggingFramer{
			Base: framer,
		}
	}
	conn, err := jsonrpc2.Dial(
		ctx,
		dialer,
		jsonrpc2.ConnectionOptions{
			Handler: logHandler(),
			Framer:  framer,
		},
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("dial error: %w", err)
	}

	return &MCPClient{
		conn:     conn,
		cancelFn: cancel,
	}, nil
}

type ServerInfo InitializeResult

// Initialize sends the initialize request to the server and stores the capabilities
func (c *MCPClient) Initialize(ctx context.Context) (*ServerInfo, error) {
	method := "initialize"
	params := InitializeRequestParams{
		ClientInfo: Implementation{
			Name:    "mcptest",
			Version: "0.1.0",
		},
		ProtocolVersion: "2024-11-05",
		Capabilities:    ClientCapabilities{
			// Add capabilities as needed
		},
	}

	var result InitializeResult
	if err := c.conn.Call(ctx, method, params).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	c.ServerInfo = (*ServerInfo)(&result)
	c.initialized = true

	log.Printf(
		"Server initialized: %s %s",
		c.ServerInfo.ServerInfo.Name,
		c.ServerInfo.ServerInfo.Version,
	)
	if c.ServerInfo.Instructions != nil {
		log.Printf("Server instructions: %s", *c.ServerInfo.Instructions)
	}

	for k, v := range c.ServerInfo.Capabilities.Logging {
		fmt.Printf("Logging: %s: %v\n", k, v)
	}

	// Send initialized notification
	if err := c.conn.Notify(ctx, "notifications/initialized", nil); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}
	return c.ServerInfo, nil
}

// Ping sends a ping request to check if the server is alive
func (c *MCPClient) Ping(ctx context.Context) error {
	if err := c.conn.Call(ctx, "ping", nil).Await(ctx, nil); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// ListTools requests the list of available tools from the server
func (c *MCPClient) ListTools(ctx context.Context, cursor *string) ([]Tool, *string, error) {
	params := &ListToolsRequestParams{Cursor: cursor}

	var result ListToolsResult
	if err := c.conn.Call(ctx, "tools/list", params).Await(ctx, &result); err != nil {
		return nil, nil, fmt.Errorf("list tools failed: %w", err)
	}

	return result.Tools, nil, nil
}

// ListResources requests the list of available resources from the server
func (c *MCPClient) ListResources(
	ctx context.Context,
	cursor *string,
) ([]Resource, *string, error) {
	params := &ListResourcesRequestParams{Cursor: cursor}

	var result ListResourcesResult
	if err := c.conn.Call(ctx, "resources/list", params).Await(ctx, &result); err != nil {
		return nil, nil, fmt.Errorf("list resources failed: %w", err)
	}

	return result.Resources, result.NextCursor, nil
}

// ReadResource reads a specific resource from the server
func (c *MCPClient) ReadResource(
	ctx context.Context,
	uri string,
) (*[]interface{}, error) {
	var result ReadResourceResult
	params := ReadResourceRequestParams{Uri: uri}
	if err := c.conn.Call(ctx, "resources/read", params).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("read resource failed: %w", err)
	}

	return &result.Contents, nil
}

// CallTool executes a specific tool with given parameters
func (c *MCPClient) CallTool(
	ctx context.Context,
	name string,
	args map[string]interface{},
) (*CallToolResult, error) {
	params := CallToolRequestParams{
		Name:      name,
		Arguments: args,
	}
	var result CallToolResult
	if err := c.conn.Call(ctx, "tools/call", params).Await(ctx, &result); err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return &result, nil
}

// Close shuts down the MCP client and server
func (c *MCPClient) Close() error {
	ctx := context.Background()

	// Send exit notification
	if err := c.conn.Notify(ctx, "exit", nil); err != nil {
		log.Printf("exit notification failed: %v", err)
	}

	// Close the connection
	if err := c.conn.Close(); err != nil {
		// log.Printf("connection close failed: %v", err)
	}

	// Cancel the context and wait for the process to finish
	c.cancelFn()

	return nil
}
