package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func createTask(c *gin.Context) {
	callerMethod := "createTask"
	startTime := time.Now()
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()
	validationStatus, jobCount := validateUserCounts(c)
	log(callerMethod, fmt.Sprintf("Validation status: %v", validationStatus))

	// Parse and validate request body
	input, err := parseRequestBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate Task ID
	taskID, err := generateTaskID(c, jobCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate task ID"})
		return
	}

	timeUTC, err := time.Parse("2006-01-02 15:04:05", input.StartFrom)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Time is in UTC and needs to be in the format: 2000-12-02 01:01:01"})
		return
	}

	userId := c.GetString("userId")
	log(callerMethod, userId)
	log(callerMethod, taskID)
	// Create Task struct
	task := createTaskStruct(input, taskID, userId)

	// Append task to file
	// appendTaskToFile(task)
	err = appendTaskToDynamoDB(task)

	if err != nil {
		log(callerMethod, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert into db"})
		return
	}

	updatetJobCount(c, jobCount+1)

	// Create Job and add to heap
	job := Job{taskID, timeUTC.Unix()}
	go addToHeap(job)

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

func generateTaskID(c *gin.Context, jobCount int64) (string, error) {
	callerMethod := "generateTaskID"
	// Fetch userID and jobCount from the context
	userID := c.GetString("userId")

	jobCount += 1
	log(callerMethod, fmt.Sprintf("Updated count %d", jobCount))

	// Construct the task ID
	taskID := fmt.Sprintf("%s_%d", userID, jobCount)
	log(callerMethod, taskID)

	return taskID, nil
}

func createTaskStruct(input CreateTaskInput, taskID string, userId string) Task {
	lastExecution := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)
	timeUTC, _ := time.Parse("2006-01-02 15:04:05", input.StartFrom)
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
		UserID:              userId,
		APIBody:             input.APIBody,
		NextExecution:       timeUTC,
	}
}

//DO NOT DELETE!
// func appendTaskToFile(task Task) error {
// 	// Open or create the data file
// 	file, err := os.OpenFile(dataFile, os.O_RDWR|os.O_CREATE, 0644)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	// Decode existing tasks from file
// 	var tasks []Task
// 	if err := json.NewDecoder(file).Decode(&tasks); err != nil && err != io.EOF {
// 		return err
// 	}

// 	// Append the new task
// 	tasks = append(tasks, task)

// 	// Seek to the beginning of the file to overwrite its contents
// 	file.Seek(0, 0)

// 	// Write the tasks back to the file
// 	if err := json.NewEncoder(file).Encode(tasks); err != nil {
// 		return err
// 	}

// 	return nil
// }
