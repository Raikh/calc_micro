package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Helper function to create a test request
func createTestRequest(method, path string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Setup function to initialize test environment
func setupTest() {
	expressions = make(map[string]*Expression)
	tasks = make(map[string]*Task)
}

func TestHandleCalculate(t *testing.T) {
	setupTest()

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:           "Valid expression",
			requestBody:    `{"expression": "2 + 3 * 4"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   true,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"expression": "2 + 3 * 4" -- invalid`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   false,
		},
		{
			name:           "Empty expression",
			requestBody:    `{"expression": ""}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest("POST", "/api/v1/calculate", tt.requestBody)
			rr := httptest.NewRecorder()

			handleCalculate(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedBody {
				var response map[string]string
				err := json.NewDecoder(rr.Body).Decode(&response)
				if err != nil {
					t.Errorf("Failed to decode response body: %v", err)
				}
				if _, ok := response["id"]; !ok {
					t.Error("Expected id in response, but got none")
				}
			}
		})
	}
}

func TestHandleGetExpressions(t *testing.T) {
	setupTest()

	// Add test data
	expressions["test1"] = &Expression{
		ID:     "test1",
		Expr:   "2 + 3",
		Status: "completed",
		Result: 5,
	}

	req := createTestRequest("GET", "/api/v1/expressions", "")
	rr := httptest.NewRecorder()

	handleGetExpressions(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	exprs, ok := response["expressions"].([]interface{})
	if !ok {
		t.Fatal("Expected expressions array in response")
	}

	if len(exprs) != 1 {
		t.Errorf("Expected 1 expression, got %d", len(exprs))
	}
}

func TestHandleGetExpressionByID(t *testing.T) {
	setupTest()

	// Add test data
	expressions["test1"] = &Expression{
		ID:     "test1",
		Expr:   "2 + 3",
		Status: "completed",
		Result: 5,
	}

	tests := []struct {
		name           string
		expressionID   string
		expectedStatus int
	}{
		{
			name:           "Existing expression",
			expressionID:   "test1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existing expression",
			expressionID:   "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest("GET", "/api/v1/expressions/"+tt.expressionID, "")
			rr := httptest.NewRecorder()

			handleGetExpressionByID(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(rr.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}

				expr, ok := response["expression"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected expression object in response")
				}

				if id, ok := expr["id"].(string); !ok || id != tt.expressionID {
					t.Errorf("Expected expression ID %s, got %v", tt.expressionID, id)
				}
			}
		})
	}
}

func TestHandleTask(t *testing.T) {
	setupTest()

	// Add test data
	testTask := &Task{
		ID:            "task1",
		ExpressionID:  "expr1",
		Arg1:          2,
		Arg2:          3,
		Operation:     "+",
		OperationTime: 1000,
		Dependencies:  []string{},
		Completed:     false,
		IsProcessing:  false,
	}
	tasks["task1"] = testTask

	expressions["expr1"] = &Expression{
		ID:     "expr1",
		Expr:   "2 + 3",
		Status: "pending",
	}

	tests := []struct {
		name           string
		method         string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Get available task",
			method:         "GET",
			requestBody:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Submit result",
			method:         "POST",
			requestBody:    `{"id": "task1", "result": 5}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid task result",
			method:         "POST",
			requestBody:    `{"id": "nonexistent", "result": 5}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(tt.method, "/internal/task", tt.requestBody)
			rr := httptest.NewRecorder()

			handleTask(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.method == "GET" && tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(rr.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response body: %v", err)
				}

				if _, ok := response["task"]; !ok {
					t.Error("Expected task in response, but got none")
				}
			}
		})
	}
}

func TestParseExpression(t *testing.T) {
	tests := []struct {
		name          string
		expression    string
		expectedTasks int
	}{
		{
			name:          "Simple addition",
			expression:    "2 + 3",
			expectedTasks: 1,
		},
		{
			name:          "Complex expression",
			expression:    "2 + 3 * 4",
			expectedTasks: 2,
		},
		{
			name:          "Expression with parentheses",
			expression:    "(2 + 3) * 4",
			expectedTasks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := parseExpression(tt.expression, "test-expr-id")
			if len(tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTasks, len(tasks))
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected []string
	}{
		{
			name:     "Simple expression",
			expr:     "2 + 3",
			expected: []string{"2", "+", "3"},
		},
		{
			name:     "Complex expression",
			expr:     "2 + 3 * 4",
			expected: []string{"2", "+", "3", "*", "4"},
		},
		{
			name:     "Expression with parentheses",
			expr:     "(2 + 3) * 4",
			expected: []string{"(", "2", "+", "3", ")", "*", "4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.expr)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(result))
				return
			}
			for i, token := range result {
				if token != tt.expected[i] {
					t.Errorf("Expected token %s at position %d, got %s", tt.expected[i], i, token)
				}
			}
		})
	}
}
