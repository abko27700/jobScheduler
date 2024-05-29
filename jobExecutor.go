package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type JobExecutionResult struct {
	Status      string        // Status of the job execution (success or failure)
	Error       error         // Any error encountered during execution
	ElapsedTime time.Duration // Time taken to execute the job
}

func jobExecutor(jobId string) bool {
	startTime := time.Now()
	callerMethod := "jobExecutor"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	if isTaskDeleted(jobId) {
		log(callerMethod, fmt.Sprintf("Not executing since jobId:%s is deleted.", jobId))
		return true
	}

	task, err := getTaskFromDB(jobId)
	if err != nil {
		log(callerMethod, err.Error())
		return false
	}

	log(callerMethod, fmt.Sprintf("Task API URL: %s", task.APIURL))
	log(callerMethod, fmt.Sprintf("Executing jobId:%s", jobId))
	if task.APIMethod == "POST" {
		executePOSTRequest(*task)
	}
	task.LastExecution = time.Now()
	task.TotalExecutions += 1
	if task.TotalExecutions != 30 {
		nextExecution := time.Now().Add(time.Duration(task.Frequency) * time.Second)
		newJob := Job{
			ID:   task.TaskID,
			Time: nextExecution.Unix(),
		}
		addToHeap(newJob)
		task.NextExecution = nextExecution
	}
	updateTaskInDb(task)

	return true
}

//DO NOT DELETE!!
// Should create the task object for the executable
// func getTaskFromJson(taskId string) Task {
// 	startTime := time.Now()
// 	callerMethod := "jobExecutor"
// 	log(callerMethod, "Start")

// 	defer func() {
// 		endLog(callerMethod, startTime)
// 	}()

// 	data, err := os.ReadFile("data.json")
// 	if err != nil {
// 		fmt.Println("Error reading file:", err)
// 		return Task{}
// 	}

// 	// Unmarshal JSON data into a slice of tasks
// 	var tasks []Task
// 	if err := json.Unmarshal(data, &tasks); err != nil {
// 		fmt.Println("Error unmarshalling JSON:", err)
// 		return Task{}
// 	}

// 	// Find the task with the given ID
// 	for _, t := range tasks {
// 		if t.TaskID == taskId {
// 			return t
// 		}
// 	}

// 	return Task{} // Task not found
// }

func executePOSTRequest(task Task) JobExecutionResult {
	startTime := time.Now()
	callerMethod := "executePOSTRequest"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	result := JobExecutionResult{} // Initialize the result struct

	if task.APIMethod != "POST" {
		// Set status to failure and return
		result.Status = "failure"
		result.Error = errors.New("API method is not POST")
		result.ElapsedTime = time.Since(startTime)
		return result
	}

	// Marshal the API body into a JSON string
	reqBodyJSON, err := json.Marshal(task.APIBody)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Errorf("error marshalling API body: %v", err)
		result.ElapsedTime = time.Since(startTime)
		return result
	}

	// Create a new buffer with the JSON string
	reqBody := bytes.NewBuffer(reqBodyJSON)

	// Create the HTTP request
	req, err := http.NewRequest("POST", task.APIURL, reqBody)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Errorf("error creating request: %v", err)
		result.ElapsedTime = time.Since(startTime)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Errorf("error making request: %v", err)
		result.ElapsedTime = time.Since(startTime)
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Errorf("error reading response body: %v", err)
		result.ElapsedTime = time.Since(startTime)
		return result
	}

	// Print the response status and body
	log(callerMethod, fmt.Sprintf("Response Status: %s", resp.Status))
	log(callerMethod, fmt.Sprintf("Response Body: %s", body))

	result.Status = "success"
	result.ElapsedTime = time.Since(startTime)
	return result
}
