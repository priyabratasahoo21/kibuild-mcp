package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	pluginConn   net.Conn
	connMu       sync.RWMutex
	pendingCalls = make(map[string]chan *RPCResponse)
	callsMu      sync.Mutex
	idCounter    uint64
)

type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      string      `json:"id"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      string          `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func SetPluginConn(c net.Conn) {
	connMu.Lock()
	defer connMu.Unlock()
	pluginConn = c
}

func GetPluginConn() net.Conn {
	connMu.RLock()
	defer connMu.RUnlock()
	return pluginConn
}

func IsPluginConnected() bool {
	return GetPluginConn() != nil
}

// HandleIncomingRPC processes responses from the C++ plugin
func HandleIncomingRPC(data []byte) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}

	if resp.ID != "" {
		callsMu.Lock()
		ch, ok := pendingCalls[resp.ID]
		if ok {
			delete(pendingCalls, resp.ID)
			callsMu.Unlock()
			ch <- &resp
			return
		}
		callsMu.Unlock()
	}
}

func CallPlugin(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	conn := GetPluginConn()
	if conn == nil {
		return nil, errors.New("no active FileMaker plugin connection")
	}

	callsMu.Lock()
	idCounter++
	id := fmt.Sprintf("req_%d", idCounter)
	respChan := make(chan *RPCResponse, 1)
	pendingCalls[id] = respChan
	callsMu.Unlock()

	req := RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		callsMu.Lock()
		delete(pendingCalls, id)
		callsMu.Unlock()
		return nil, err
	}

	// Append newline as message delimiter for JSON-RPC over TCP/Unix sockets
	reqBytes = append(reqBytes, '\n')

	if _, err := conn.Write(reqBytes); err != nil {
		callsMu.Lock()
		delete(pendingCalls, id)
		callsMu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		callsMu.Lock()
		delete(pendingCalls, id)
		callsMu.Unlock()
		return nil, ctx.Err()
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("plugin error (%d): %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func ExportSchema(dbName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := map[string]string{"database": dbName}
	result, err := CallPlugin(ctx, "export_schema", params)
	if err != nil {
		return "", err
	}

	var resStr string
	if err := json.Unmarshal(result, &resStr); err != nil {
		return string(result), nil
	}
	return resStr, nil
}

func ReadLayout(layoutName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	params := map[string]string{"layout_name": layoutName}
	result, err := CallPlugin(ctx, "read_layout", params)
	if err != nil {
		return "", err
	}

	var resStr string
	if err := json.Unmarshal(result, &resStr); err != nil {
		return string(result), nil
	}
	return resStr, nil
}

func GetActiveContext() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := CallPlugin(ctx, "get_active_context", nil)
	if err != nil {
		return "", err
	}

	var resStr string
	if err := json.Unmarshal(result, &resStr); err != nil {
		return string(result), nil
	}
	return resStr, nil
}
