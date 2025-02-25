package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Mock server for testing
func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/task":
			if r.Method == "GET" {
				task := Task{
					ID:            "test-123",
					Arg1:          10,
					Arg2:          5,
					Operation:     "+",
					OperationTime: 100,
				}
				json.NewEncoder(w).Encode(map[string]Task{"task": task})
			} else if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestGetTask(t *testing.T) {
	// Setup test server
	server := setupTestServer()
	defer server.Close()

	// Set the API base URL to our test server
	api_base_url = server.URL

	// Test getting a task
	task := getTask()
	if task == nil {
		t.Fatal("Expected task, got nil")
	}

	if task.ID != "test-123" {
		t.Errorf("Expected task ID 'test-123', got %s", task.ID)
	}

	if task.Operation != "+" {
		t.Errorf("Expected operation '+', got %s", task.Operation)
	}
}

func TestComputeTask(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected float64
	}{
		{
			name: "Addition",
			task: Task{
				Arg1:          10,
				Arg2:          5,
				Operation:     "+",
				OperationTime: 0,
			},
			expected: 15,
		},
		{
			name: "Subtraction",
			task: Task{
				Arg1:          10,
				Arg2:          5,
				Operation:     "-",
				OperationTime: 0,
			},
			expected: 5,
		},
		{
			name: "Multiplication",
			task: Task{
				Arg1:          10,
				Arg2:          5,
				Operation:     "*",
				OperationTime: 0,
			},
			expected: 50,
		},
		{
			name: "Division",
			task: Task{
				Arg1:          10,
				Arg2:          5,
				Operation:     "/",
				OperationTime: 0,
			},
			expected: 2,
		},
		{
			name: "Invalid Operation",
			task: Task{
				Arg1:          10,
				Arg2:          5,
				Operation:     "invalid",
				OperationTime: 0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeTask(&tt.task)
			if result != tt.expected {
				t.Errorf("computeTask() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSendResult(t *testing.T) {
	// Setup test server
	server := setupTestServer()
	defer server.Close()

	// Set the API base URL to our test server
	api_base_url = server.URL

	// Test sending a result
	sendResult("test-123", 15.0)
	// If we reach here without panic, the test passes
	// In a real scenario, you might want to verify the request body
}

func TestWorkerIntegration(t *testing.T) {
	// Setup test server
	server := setupTestServer()
	defer server.Close()

	// Set the API base URL to our test server
	api_base_url = server.URL

	// Start worker in a goroutine
	done := make(chan bool)
	go func() {
		worker()
		done <- true
	}()

	// Let the worker run for a short time
	time.Sleep(200 * time.Millisecond)

	// Test passes if we reach here without any panics
	// In a real scenario, you might want to verify the complete flow
}

func TestInitBaseUrl(t *testing.T) {
	// Test cases
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "Default URL",
			args:     []string{},
			expected: "http://127.0.0.1:8080",
		},
		{
			name:     "Custom URL",
			args:     []string{"-base-url", "http://localhost:9090"},
			expected: "http://localhost:9090",
		},
		{
			name:     "Custom IP and Port",
			args:     []string{"-base-url", "http://192.168.1.100:3000"},
			expected: "http://192.168.1.100:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original flags
			oldFlags := flag.CommandLine
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Save original args and restore them after the test
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
				flag.CommandLine = oldFlags
			}()

			// Set up test args
			os.Args = append([]string{"cmd"}, tt.args...)

			// Reset api_base_url before each test
			api_base_url = ""

			// Call the function
			initBaseUrl()

			// Check the result
			if api_base_url != tt.expected {
				t.Errorf("initBaseUrl() got = %v, want %v", api_base_url, tt.expected)
			}
		})
	}
}
