package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Expression struct {
	ID     string
	Expr   string
	Status string
	Result float64
}

type Task struct {
	ID            string
	ExpressionID  string
	Arg1          float64
	Arg2          float64
	Operation     string
	OperationTime int      `json:"operation_time"`
	Dependencies  []string `json:"-"`
	Result        float64  `json:"-"`
	Completed     bool     `json:"-"`
	IsProcessing  bool     `json:"-"`
}

var (
	expressions            = make(map[string]*Expression)
	tasks                  = make(map[string]*Task)
	mutex                  = &sync.Mutex{}
	time_addition_ms       = getEnvAsInt("TIME_ADDITION_MS", 1000)
	time_subtraction_ms    = getEnvAsInt("TIME_SUBTRACTION_MS", 1000)
	time_multiplication_ms = getEnvAsInt("TIME_MULTIPLICATIONS_MS", 2000)
	time_division_ms       = getEnvAsInt("TIME_DIVISIONS_MS", 2000)
)

func initListenAddress() string {
	var ipaddress_string, port_string string
	flag.StringVar(&ipaddress_string, "ip", "127.0.0.1", "Listen on IP address")
	flag.StringVar(&port_string, "port", "8080", "Listen on port")
	flag.Parse()

	if ipaddress_string == "*" {
		ipaddress_string = ""
	}
	return ipaddress_string + ":" + port_string
}
func main() {

	http.HandleFunc("/api/v1/calculate", handleCalculate)
	http.HandleFunc("/api/v1/expressions", handleGetExpressions)
	http.HandleFunc("/api/v1/expressions/", handleGetExpressionByID)
	http.HandleFunc("/internal/task", handleTask)

	log.Fatal(http.ListenAndServe(initListenAddress(), nil))
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}
	var req struct {
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusUnprocessableEntity)
		return
	}

	req.Expression = strings.TrimSpace(req.Expression)
	if req.Expression == "" {
		http.Error(w, "Invalid request body", http.StatusUnprocessableEntity)
		return
	}

	id := generateID()
	expr := &Expression{
		ID:     id,
		Expr:   req.Expression,
		Status: "pending",
	}

	mutex.Lock()
	expressions[id] = expr
	mutex.Unlock()

	tasksForExpr := parseExpression(req.Expression, id)
	for _, task := range tasksForExpr {
		mutex.Lock()
		tasks[task.ID] = task
		mutex.Unlock()
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handleGetExpressions(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	var exprs []map[string]interface{}
	for _, expr := range expressions {
		exprs = append(exprs, map[string]interface{}{
			"id":     expr.ID,
			"status": expr.Status,
			"result": expr.Result,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"expressions": exprs})
}

func handleGetExpressionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/expressions/"):]

	mutex.Lock()
	expr, exists := expressions[id]
	mutex.Unlock()

	if !exists {
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"expression": map[string]interface{}{
			"id":     expr.ID,
			"status": expr.Status,
			"result": expr.Result,
		},
	})
}

func handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		mutex.Lock()
		for _, task := range tasks {
			if task.IsProcessing {
				continue
			}
			if !task.Completed && areDependenciesCompleted(task) {
				json.NewEncoder(w).Encode(map[string]interface{}{"task": task})
				task.IsProcessing = true
				mutex.Unlock()
				return
			}
		}
		mutex.Unlock()
		http.Error(w, "No tasks available", http.StatusNotFound)
	} else if r.Method == http.MethodPost {
		var req struct {
			ID     string  `json:"id"`
			Result float64 `json:"result"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusUnprocessableEntity)
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		task, exists := tasks[req.ID]
		if !exists {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		task.Result = req.Result
		task.Completed = true

		expr, exists := expressions[task.ExpressionID]
		if !exists {
			http.Error(w, "Expression not found", http.StatusNotFound)
			return
		}

		if isFinalTask(task.ExpressionID) {
			expr.Result = task.Result
			expr.Status = "completed"
			clearExpressionTasks(task.ExpressionID)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func clearExpressionTasks(expressionID string) {
	for id, task := range tasks {
		if task.ExpressionID == expressionID {
			delete(tasks, id)
		}
	}
}

func areDependenciesCompleted(task *Task) bool {
	for idx, depID := range task.Dependencies {
		depTask, exists := tasks[depID]
		if !exists || !depTask.Completed {
			return false
		}
		updateTaskByDependency(task, idx, depTask.Result)
	}
	return true
}

func updateTaskByDependency(task *Task, index int, value float64) {
	depsCount := len(task.Dependencies)
	if depsCount == 1 {
		if task.Arg1 == 0 && task.Arg2 != 0 {
			task.Arg1 = value
		} else {
			task.Arg2 = value
		}
	} else if depsCount == 2 {
		if index == 0 {
			task.Arg1 = value
		} else {
			if task.Arg1 == 0 && task.Arg2 != 0 {
				task.Arg1 = value
			} else {
				task.Arg2 = value
			}
		}
	}
}

func isFinalTask(expressionID string) bool {
	for _, task := range tasks {
		if task.ExpressionID == expressionID && !task.Completed {
			return false
		}
	}
	return true
}

func generateID() (uuid string) {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

func parseExpression(expr string, expressionID string) []*Task {
	postfix := infixToPostfix(expr)
	stack := []*Task{}
	tasks := []*Task{}

	for _, token := range postfix {
		switch token {
		case "+", "-", "*", "/":
			arg2Task := stack[len(stack)-1]
			arg1Task := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			deps := []string{}
			if !arg1Task.Completed {
				deps = append(deps, arg1Task.ID)
			}
			if !arg2Task.Completed {
				deps = append(deps, arg2Task.ID)
			}
			task := &Task{
				ID:            generateID(),
				ExpressionID:  expressionID,
				Arg1:          arg1Task.Result,
				Arg2:          arg2Task.Result,
				Operation:     token,
				OperationTime: getOperationTime(token),
				Dependencies:  deps,
			}
			tasks = append(tasks, task)

			stack = append(stack, task)
		default:
			num, _ := strconv.ParseFloat(token, 64)
			task := &Task{
				ID:            generateID(),
				ExpressionID:  expressionID,
				Arg1:          num,
				Arg2:          0,
				Operation:     "",
				OperationTime: 0,
				Dependencies:  []string{},
				Result:        num,
				Completed:     true,
			}
			stack = append(stack, task)
		}
	}

	return tasks
}

func infixToPostfix(expr string) []string {
	var output []string
	var stack []string

	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	tokens := tokenize(expr)
	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			for len(stack) > 0 && precedence[stack[len(stack)-1]] >= precedence[token] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		case "(":
			stack = append(stack, token)
		case ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = stack[:len(stack)-1]
		default:
			output = append(output, token)
		}
	}

	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output
}

func tokenize(expr string) []string {
	var tokens []string
	var currentToken string

	for _, char := range expr {
		if char == ' ' {
			continue
		}
		if char == '+' || char == '-' || char == '*' || char == '/' || char == '(' || char == ')' {
			if currentToken != "" {
				tokens = append(tokens, currentToken)
				currentToken = ""
			}
			tokens = append(tokens, string(char))
		} else {
			currentToken += string(char)
		}
	}

	if currentToken != "" {
		tokens = append(tokens, currentToken)
	}

	return tokens
}

func getOperationTime(operation string) int {
	switch operation {
	case "+":
		return time_addition_ms
	case "-":
		return time_subtraction_ms
	case "*":
		return time_multiplication_ms
	case "/":
		return time_division_ms
	default:
		return 1000
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return defaultValue
	}
	return value
}
