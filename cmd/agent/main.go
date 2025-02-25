package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Task struct {
	ID            string  `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operation     string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
}

var (
	api_base_url string
)

func getTask() *Task {
	resp, err := http.Get(api_base_url + "/internal/task")
	if err != nil {
		log.Println("Error getting task:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	var taskResponse struct {
		Task Task `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&taskResponse); err != nil {
		log.Println("Error decoding task:", err)
		return nil
	}

	return &taskResponse.Task
}

func computeTask(task *Task) float64 {
	time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

	switch task.Operation {
	case "+":
		return task.Arg1 + task.Arg2
	case "-":
		return task.Arg1 - task.Arg2
	case "*":
		return task.Arg1 * task.Arg2
	case "/":
		return task.Arg1 / task.Arg2
	default:
		return 0
	}
}

func sendResult(taskID string, result float64) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"id":     taskID,
		"result": result,
	})

	resp, err := http.Post(
		api_base_url+"/internal/task",
		"application/json",
		bytes.NewBuffer(reqBody),
	)

	if err != nil {
		log.Println("Error sending result:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Error sending result:", resp.Status)
	}
}

func worker() {
	for {
		task := getTask()
		if task == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		result := computeTask(task)
		sendResult(task.ID, result)
	}
}

func initBaseUrl() {
	flag.StringVar(&api_base_url, "base-url", "http://127.0.0.1:8080", "Listen on IP address")
	flag.Parse()
}

func main() {
	initBaseUrl()
	computingPower, _ := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if computingPower == 0 {
		computingPower = 2
	}

	for i := 0; i < computingPower; i++ {
		go worker()
	}

	select {}
}
