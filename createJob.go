package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"

	// "log"

	// "log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// var (
// 	// tasksMutex    sync.Mutex
// 	taskIDCounter int
// 	taskIDMutex   sync.Mutex
// )

func createTask(c *gin.Context) {
	callerMethod := "createTask"
	startTime := time.Now()
	clientId := 1289
	log(clientId, callerMethod, "Start")
	defer func() {
		endLog(clientId, callerMethod, startTime)
	}()
	// Parse and validate request body
	input, err := parseRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate Task ID
	taskID, err := generateTaskID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate task ID"})
		return
	}

	// Create Task struct
	task := createTaskStruct(input, taskID)

	// Append task to file
	err = appendTaskToFile(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to data file"})
		return
	}

	timeUTC, err := time.Parse("2006-01-02 15:04:05", input.StartFrom)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Time is in UTC and needs to be in the format: 2000-12-02 01:01:01"})
		return
	}

	// Create Job and add to heap
	job := Job{ID: taskID, Time: timeUTC.Unix()}
	go addToHeap(job, input.UserID)

	c.JSON(http.StatusOK, gin.H{"taskId": taskID})
}

func parseRequestBody(c *gin.Context) (CreateTaskInput, error) {
	var input CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		return CreateTaskInput{}, err
	}
	if input.APIMethod == "" || input.APIURL == "" || input.StartFrom == "" {
		return CreateTaskInput{}, errors.New("apiMethod, apiURL, startFrom are required fields")
	}
	return input, nil
}

func generateTaskID() (int, error) {
	return rand.Intn(100000), nil
}

func createTaskStruct(input CreateTaskInput, taskID int) Task {
	lastExecution := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)
	// log(99, "createTaskStruct apiUrl: ", input.APIURL)
	return Task{
		TaskID:              taskID,
		LastExecution:       lastExecution,
		TotalExecutions:     0,
		APIMethod:           input.APIMethod,
		APIURL:              input.APIURL,
		AvgTimePerExecution: 0,
		TimeOutAfter:        0,
		StartFrom:           input.StartFrom,
		Frequency:           input.Frequency,
		UserID:              input.UserID,
		APIBody:             input.APIBody,
	}
}

func appendTaskToFile(task Task) error {
	// Open or create the data file
	file, err := os.OpenFile(dataFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode existing tasks from file
	var tasks []Task
	if err := json.NewDecoder(file).Decode(&tasks); err != nil && err != io.EOF {
		return err
	}

	// Append the new task
	tasks = append(tasks, task)

	// Seek to the beginning of the file to overwrite its contents
	file.Seek(0, 0)

	// Write the tasks back to the file
	if err := json.NewEncoder(file).Encode(tasks); err != nil {
		return err
	}

	return nil
}

// This function ensures that the saved jobs are loaded into the heap. Currently loading them using json. Later will load using dynamoDB.
func loadExistingJobs() {
	// Read data from the data.json file
	data, err := os.ReadFile("data.json")
	if err != nil {
		fmt.Println("Error reading file:", err)
	}

	// Unmarshal JSON data into a slice of tasks
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// Find the task with the given ID
	for _, task := range tasks {
		newJob := Job{ID: task.TaskID, Time: task.LastExecution.Unix()}
		addToHeap(newJob, task.UserID)

		// Send the job to addToHeap function
		addToHeap(newJob, 1289) // Pass clientId or use a default value
	}
}
