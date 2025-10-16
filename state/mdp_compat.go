package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
)

// MDPCompatibilityHandler wraps gRPC service for MDP compatibility
type MDPCompatibilityHandler struct {
	grpcClient statev1connect.StateServiceClient
}

func NewMDPCompatibilityHandler(grpcClient statev1connect.StateServiceClient) *MDPCompatibilityHandler {
	return &MDPCompatibilityHandler{grpcClient: grpcClient}
}

// HandleMDPRequest processes MDP-style requests and converts to gRPC
func (h *MDPCompatibilityHandler) HandleMDPRequest(command string, args []string) ([]string, error) {
	ctx := context.Background()

	switch command {
	case "get":
		if len(args) < 1 {
			return nil, fmt.Errorf("get requires key argument")
		}

		scope := "default"
		key := args[0]

		// Support optional scope as second argument
		if len(args) > 1 {
			scope = args[1]
		}

		req := &statev1.GetRequest{
			Key:   key,
			Scope: scope,
		}

		resp, err := h.grpcClient.Get(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}

		return []string{resp.Msg.Value}, nil

	case "set":
		if len(args) < 2 {
			return nil, fmt.Errorf("set requires key and value arguments")
		}

		scope := "default"
		key := args[0]
		value := args[1]

		// Support optional scope as third argument
		if len(args) > 2 {
			scope = args[2]
		}

		req := &statev1.SetRequest{
			Key:   key,
			Value: value,
			Scope: scope,
		}

		_, err := h.grpcClient.Set(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}

		return []string{"OK"}, nil

	case "delete":
		if len(args) < 1 {
			return nil, fmt.Errorf("delete requires key argument")
		}

		scope := "default"
		key := args[0]

		// Support optional scope as second argument
		if len(args) > 1 {
			scope = args[1]
		}

		req := &statev1.DeleteRequest{
			Key:   key,
			Scope: scope,
		}

		resp, err := h.grpcClient.Delete(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}

		if resp.Msg.Existed {
			return []string{"DELETED"}, nil
		}
		return []string{"NOT_FOUND"}, nil

	case "list":
		scope := "default"
		prefix := ""

		// Support optional prefix and scope arguments
		if len(args) > 0 {
			prefix = args[0]
		}
		if len(args) > 1 {
			scope = args[1]
		}

		req := &statev1.ListRequest{
			Prefix:        prefix,
			Scope:         scope,
			IncludeValues: false, // For MDP compatibility, just return keys
		}

		stream, err := h.grpcClient.List(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}

		var keys []string
		for stream.Receive() {
			msg := stream.Msg()
			keys = append(keys, msg.Key)
		}

		if err := stream.Err(); err != nil {
			return nil, err
		}

		return keys, nil

	case "heartbeat":
		// MDP heartbeat -> gRPC health check
		req := &emptypb.Empty{}

		resp, err := h.grpcClient.Health(ctx, connect.NewRequest(req))
		if err != nil {
			return []string{"UNHEALTHY"}, err
		}

		if resp.Msg.Status == healthv1.HealthCheckResponse_SERVING_STATUS_SERVING {
			return []string{"HEALTHY"}, nil
		}
		return []string{"UNHEALTHY"}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}

// HandleMDPJSON processes JSON-formatted MDP requests for backward compatibility
func (h *MDPCompatibilityHandler) HandleMDPJSON(jsonData string) (string, error) {
	var request map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &request); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}

	commandInterface, ok := request["command"]
	if !ok {
		return "", fmt.Errorf("missing command field")
	}

	command, ok := commandInterface.(string)
	if !ok {
		return "", fmt.Errorf("command must be a string")
	}

	argsInterface, ok := request["args"]
	if !ok {
		argsInterface = []interface{}{}
	}

	argsList, ok := argsInterface.([]interface{})
	if !ok {
		return "", fmt.Errorf("args must be an array")
	}

	// Convert interface{} args to string args
	var args []string
	for _, arg := range argsList {
		if str, ok := arg.(string); ok {
			args = append(args, str)
		} else {
			args = append(args, fmt.Sprintf("%v", arg))
		}
	}

	// Handle the request
	result, err := h.HandleMDPRequest(command, args)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		jsonBytes, _ := json.Marshal(response)
		return string(jsonBytes), err
	}

	response := map[string]interface{}{
		"success": true,
		"result":  result,
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes), nil
}

// ConvertMDPRequestToHTTP creates an HTTP handler for MDP compatibility
func (h *MDPCompatibilityHandler) ConvertMDPRequestToHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse command and args from different sources
		var command string
		var args []string

		// Try to get from URL path first (e.g., /mdp/get/key1)
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) >= 2 && pathParts[0] == "mdp" {
			command = pathParts[1]
			args = pathParts[2:]
		} else {
			// Try to get from query parameters
			command = r.URL.Query().Get("command")
			if command == "" {
				http.Error(w, "Missing command", http.StatusBadRequest)
				return
			}

			// Get args from query parameter
			argsParam := r.URL.Query().Get("args")
			if argsParam != "" {
				args = strings.Split(argsParam, ",")
			}
		}

		// Handle the MDP request
		result, err := h.HandleMDPRequest(command, args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return JSON response
		response := map[string]interface{}{
			"success": true,
			"result":  result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
