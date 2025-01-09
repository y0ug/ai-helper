package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	conn   *jsonrpc2.Conn
	nextID int64
	mu     sync.Mutex
}

type InitializeParams struct {
	ProcessID             int                `json:"processId"`
	RootURI              string             `json:"rootUri"`
	Capabilities         ClientCapabilities `json:"capabilities"`
	WorkspaceFolders     []WorkspaceFolder  `json:"workspaceFolders"`
}

type ClientCapabilities struct {
	TextDocument struct {
		DocumentSymbol struct {
			DynamicRegistration bool     `json:"dynamicRegistration"`
			SymbolKind         struct{} `json:"symbolKind"`
		} `json:"documentSymbol"`
	} `json:"textDocument"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type DocumentSymbolParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

type SymbolInformation struct {
	Name     string `json:"name"`
	Kind     int    `json:"kind"`
	Location struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"start"`
		} `json:"range"`
	} `json:"location"`
	ContainerName string `json:"containerName,omitempty"`
}

func NewClient(cmd *exec.Cmd) (*Client, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	stream := jsonrpc2.NewBufferedStream(struct {
		json.Encoder
		json.Decoder
	}{
		Encoder: *json.NewEncoder(stdin),
		Decoder: *json.NewDecoder(stdout),
	}, jsonrpc2.VSCodeObjectCodec{})

	conn := jsonrpc2.NewConn(context.Background(), stream, jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (interface{}, error) {
		return nil, nil
	}))

	return &Client{conn: conn}, nil
}

func (c *Client) Initialize(ctx context.Context, rootURI string) error {
	params := InitializeParams{
		ProcessID: 1,
		RootURI:   rootURI,
		Capabilities: ClientCapabilities{},
		WorkspaceFolders: []WorkspaceFolder{{
			URI:  rootURI,
			Name: "workspace",
		}},
	}

	var result interface{}
	if err := c.conn.Call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if err := c.conn.Notify(ctx, "initialized", nil); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	return nil
}

func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]SymbolInformation, error) {
	params := DocumentSymbolParams{}
	params.TextDocument.URI = uri

	var symbols []SymbolInformation
	if err := c.conn.Call(ctx, "textDocument/documentSymbol", params, &symbols); err != nil {
		return nil, fmt.Errorf("document symbol request failed: %w", err)
	}

	return symbols, nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	var result interface{}
	if err := c.conn.Call(ctx, "shutdown", nil, &result); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	if err := c.conn.Notify(ctx, "exit", nil); err != nil {
		return fmt.Errorf("exit notification failed: %w", err)
	}

	return nil
}
