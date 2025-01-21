package mcpclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"

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
	ctx      context.Context
	logger   *slog.Logger

	// Track initialization state
	initialized bool

	// Server capabilities received during initialization
	ServerInfo *ServerInfo

	cmd    *exec.Cmd
	Stream *Stream
}

type Stream struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
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

func logHandler(logger *slog.Logger) jsonrpc2.HandlerFunc {
	return func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
		logger.Info("Request received",
			"method", req.Method,
			"id", req.ID.Raw(),
			"params", string(req.Params))
		return nil, jsonrpc2.ErrNotHandled
	}
}

type FatalServerError struct {
	Msg string
}

func (e *FatalServerError) Error() string {
	return e.Msg
}

// NewMCPClient creates a new MCP client and starts the language server
func NewMCPClient(
	ctxParent context.Context,
	logger *slog.Logger,
	serverCmd string,
	args ...string,
) (MCPClientInterface, error) {
	cmd := exec.Command(serverCmd, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	// errChan := make(chan error, 1)
	//
	// // Create a goroutine to monitor stderr
	// go func() {
	// 	errBuf := make([]byte, 4096)
	// 	for {
	// 		n, err := stderr.Read(errBuf)
	// 		if n > 0 {
	// 			logger.Error("server error", "error", string(errBuf[:n]))
	// 			errChan <- fmt.Errorf("fatal error detected: %s", string(errBuf[:n]))
	// 		}
	// 		if err != nil {
	// 			break
	// 		}
	// 	}
	// }()

	// // Wait for a short time to check if the process failed immediately
	// done := make(chan error, 1)
	// go func() {
	// 	done <- cmd.Wait()
	// }()
	//
	// // Wait for potential immediate errors
	// select {
	// case err := <-errChan:
	// 	return nil, err
	// case <-time.After(500 * time.Millisecond):
	// 	// Check if process is still running
	// 	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
	// 		return nil, fmt.Errorf("process exited prematurely")
	// 	}
	// }
	//
	// select {
	// case err := <-done:
	// 	return nil, fmt.Errorf("server process failed to start: %w", err)
	// case <-time.After(1000 * time.Millisecond):
	// 	// Process survived initial startup
	// }

	ctx, cancel := context.WithCancel(ctxParent)

	client := &MCPClient{
		cmd:      cmd,
		logger:   logger,
		ctx:      ctx,
		cancelFn: cancel,
	}
	// Start error monitoring in a goroutine
	go client.monitorErrors(stderr)

	dialer := &StdioStream{
		reader: stdout,
		writer: stdin,
	}

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
			Handler: logHandler(logger),
			Framer:  framer,
		},
	)
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return nil, fmt.Errorf("dial error: %w", err)
	}
	client.conn = conn
	return client, nil
}

func (c *MCPClient) monitorErrors(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if scanner.Scan() {
				errText := scanner.Text()
				if errText != "" {
					c.logger.Error("server error", "error", errText)

					// Only close on actual error messages
					if strings.Contains(strings.ToLower(errText), "error:") ||
						strings.Contains(
							errText,
							"BRAVE_API_KEY environment variable is required",
						) ||
						strings.Contains(strings.ToLower(errText), "fatal:") {
						c.logger.Error("fatal error detected, closing client", "error", errText)
						c.conn = nil
						c.Close()
						return
					}

					return
				}
			} else {
				// Check for scanner errors
				if err := scanner.Err(); err != nil {
					c.logger.Error("error reading stderr", "error", err)
				}
				return
			}
		}
	}
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
	c.logger.Debug("Sending initialize request")
	if err := c.conn.Call(ctx, method, params).Await(c.ctx, &result); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	c.ServerInfo = (*ServerInfo)(&result)
	c.initialized = true

	c.logger.Debug("Server initialized",
		"name", c.ServerInfo.ServerInfo.Name,
		"version", c.ServerInfo.ServerInfo.Version)
	if c.ServerInfo.Instructions != nil {
		c.logger.Debug("Server instructions", "instructions", *c.ServerInfo.Instructions)
	}

	for k, v := range c.ServerInfo.Capabilities.Logging {
		c.logger.Debug("Capabilities Logging", "key", k, "value", v)
	}

	// Send initialized notification
	if err := c.conn.Notify(ctx, "notifications/initialized", nil); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}
	return c.ServerInfo, nil
}

// Ping sends a ping request to check if the server is alive
func (c *MCPClient) Ping(ctx context.Context) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if err := c.conn.Call(ctx, "ping", nil).Await(ctx, nil); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// ListTools requests the list of available tools from the server
func (c *MCPClient) ListTools(ctx context.Context, cursor *string) ([]Tool, *string, error) {
	if !c.initialized {
		return nil, nil, fmt.Errorf("client not initialized")
	}
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
	if !c.initialized {
		return nil, nil, fmt.Errorf("client not initialized")
	}
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
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}
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
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}
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
	// _ := context.Background()
	if c.initialized {
		c.initialized = false
	}

	// If we have an active connection, clean it up
	if c.conn != nil {
		ctx := context.Background()
		// Try to send exit notification
		_ = c.conn.Notify(ctx, "exit", nil)
		// Close the connection
		_ = c.conn.Close()
		c.conn = nil
	}

	select {
	case <-c.ctx.Done():
	default:
		c.logger.Debug("Closing MCP client")
		c.cancelFn()
		// Kill the process
		if c.cmd != nil && c.cmd.Process != nil {
			if err := c.cmd.Process.Kill(); err != nil {
				c.logger.Error("failed to kill process", "error", err)
			}
		}
		// Cancel the context and wait for the process to finish

		c.logger.Debug("MCP client closed")
	}
	return nil
}
