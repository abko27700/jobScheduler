package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gin-gonic/gin"
)

var db *DynamoDBClient // Global variable to hold the DynamoDB client

// DynamoDBClient holds the DynamoDB client
type DynamoDBClient struct {
	svc *dynamodb.DynamoDB
}

// NewDynamoDBClient creates a new DynamoDB client
func NewDynamoDBClient(region string) *DynamoDBClient {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return &DynamoDBClient{
		svc: dynamodb.New(sess),
	}
}

// InitializeDb initializes the DynamoDB client
func initializeDb() {
	startTime := time.Now()
	callerMethod := "initializeDb"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()
	db = NewDynamoDBClient("us-east-1") // Specify your preferred AWS region
}

// ValidateUser checks if the user exists in the DynamoDB table
func (db *DynamoDBClient) ValidateUser(userID string) (bool, error) {
	startTime := time.Now()
	callerMethod := "ValidateUser"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()
	input := &dynamodb.GetItemInput{
		TableName: aws.String("daria_users"), // Specify your DynamoDB user table name
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
	}

	result, err := db.svc.GetItem(input)
	if err != nil {
		return false, err
	}

	return result.Item != nil, nil
}

func (d *DynamoDBClient) ValidateAPIKey(apiKey string) (string, error) {
	startTime := time.Now()
	callerMethod := "ValidateAPIKey"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	input := &dynamodb.GetItemInput{
		TableName: aws.String("daria_jobs_apiKeys"), // Specify your DynamoDB API keys table name
		Key: map[string]*dynamodb.AttributeValue{
			"APIKey": {
				S: aws.String(apiKey),
			},
		},
	}

	// Perform the GetItem operation
	result, err := d.svc.GetItem(input)

	// Check for errors
	if err != nil {
		// Log the error
		log("ValidateAPIKey", fmt.Sprintf("Error getting item: %v", err))
		return "", err
	}

	// If no item found, log and return nil
	if result.Item == nil {
		// Log no item found
		log("ValidateAPIKey", fmt.Sprintf("No item found for API Key: %s", apiKey))
		return "", nil
	}

	// Extract userID from the result
	userID := result.Item["UserID"].S

	// Log the successful operation
	log("ValidateAPIKey", fmt.Sprintf("UserID found: %s", *userID))

	// Return the userID
	return *userID, nil
}

func apiKeyAuthMiddleware(c *gin.Context) {
	startTime := time.Now()
	callerMethod := "apiKeyAuthMiddleware"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	apiKey := c.GetHeader("X-API-KEY")

	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		c.Abort()
		return
	}

	userID, err := db.ValidateAPIKey(apiKey)
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		c.Abort()
		return
	}

	log(callerMethod, fmt.Sprintf("UserID: %s", userID))

	// Store the userID in the context for further use
	c.Set("userId", userID)

	c.Next() // Pass control to the next middleware/handler
}

func validateUserCounts(c *gin.Context) (bool, int64) {
	// Retrieve the userID from the context
	userID := c.GetString("userId")

	startTime := time.Now()
	callerMethod := "validateUser"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	// Query the daria_users table to retrieve jobLimit and jobCount for the userID
	jobLimit, jobCount, err := getUserJobLimits(userID)
	if err != nil {
		log(callerMethod, fmt.Sprintf("Error: %s", err.Error()))
		return false, jobCount
	}

	log(callerMethod, fmt.Sprintf("UserID: %s, JobLimit: %d, JobCount: %d", userID, jobLimit, jobCount))
	c.Set("jobCount", jobCount)
	// Compare jobCount with jobLimit
	if jobCount >= jobLimit {
		// Return an error message if jobCount exceeds jobLimit
		errorMessage := fmt.Sprintf("Maximum job limit (%d) has been reached", jobLimit)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return false, jobCount
	}

	// User is valid

	return true, jobCount
}

func getUserJobLimits(userID string) (int64, int64, error) {
	startTime := time.Now()
	callerMethod := "getUserJobLimits"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	// Define input for GetItem operation
	input := &dynamodb.GetItemInput{
		TableName: aws.String("daria_users"),
		Key: map[string]*dynamodb.AttributeValue{
			"userId": {
				S: aws.String(userID),
			},
		},
	}

	// Execute GetItem operation
	result, err := svc.GetItem(input)
	if err != nil {
		return 0, 0, err
	}

	// Check if the item exists
	if len(result.Item) == 0 {
		return 0, 0, fmt.Errorf("user not found")
	}

	// Retrieve jobLimit and jobCount from the result
	jobLimitStr := aws.StringValue(result.Item["jobLimit"].N)
	jobCountStr := aws.StringValue(result.Item["jobCount"].N)

	// Convert strings to integers
	jobLimit, err := strconv.ParseInt(jobLimitStr, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	jobCount, err := strconv.ParseInt(jobCountStr, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return jobLimit, jobCount, nil
}

func appendTaskToDynamoDB(task Task) error {
	callerMethod := "appendTaskToDynamoDB"
	log(callerMethod, fmt.Sprintf("Task struct: %+v", task))
	// Marshal the task item into a DynamoDB attribute value
	av, err := dynamodbattribute.MarshalMap(task)
	if err != nil {
		return err
	}

	// Add the UserID to the item
	if task.UserID == "" {
		return errors.New("UserID cannot be empty")
	}
	av["UserID"] = &dynamodb.AttributeValue{S: aws.String(task.UserID)}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("daria_tasks"),
	}

	// Put the item into DynamoDB
	_, err = db.svc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

func updatetJobCount(c *gin.Context, jobCount int64) error {
	callerMethod := "updatetJobCount"
	startTime := time.Now()
	defer func() {
		endLog(callerMethod, startTime)
	}()
	log(callerMethod, "Start")

	userId := c.GetString("userId")
	// jobCount := c.GetInt("jobCount")

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("daria_users"), // Specify your DynamoDB user table name
		Key: map[string]*dynamodb.AttributeValue{
			"userId": {
				S: aws.String(userId),
			},
		},
		UpdateExpression: aws.String("SET jobCount = :c"), // Update the jobCount attribute
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":c": {
				N: aws.String(strconv.FormatInt(jobCount, 10)), // Convert jobCount to string
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"), // Return the updated attributes
	}
	result, err := db.svc.UpdateItem(input)
	if err != nil {
		log(callerMethod, err.Error())
		return err
	}

	// Log the updated job count
	log(callerMethod, fmt.Sprintf("Updated job count for user %s to %d", userId, jobCount))

	// Optional: Print the updated item
	updatedItem := map[string]interface{}{}
	err = dynamodbattribute.UnmarshalMap(result.Attributes, &updatedItem)
	if err != nil {
		log(callerMethod, fmt.Sprintf("Error unmarshalling updated attributes: %s", err.Error()))

	} else {
		log(callerMethod, fmt.Sprintf("Updated attributes: %+v", updatedItem))

	}

	return nil
}

// loadExistingJobs loads existing jobs from the DynamoDB table and adds them to the application
func loadExistingJobs() error {
	startTime := time.Now()
	callerMethod := "loadExistingJobs"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	// Define input for Scan operation
	input := &dynamodb.ScanInput{
		TableName: aws.String("daria_tasks"), // Specify your DynamoDB jobs table name
	}

	// Perform the Scan operation
	result, err := db.svc.Scan(input)
	if err != nil {
		log(callerMethod, fmt.Sprintf("Error scanning DynamoDB table: %s", err.Error()))
		return err
	}

	// Parse and add retrieved jobs to the application
	for _, item := range result.Items {
		// log(callerMethod, fmt.Sprintf("Item: %v", item))
		task := Task{}
		err := dynamodbattribute.UnmarshalMap(item, &task)
		if err != nil {
			log(callerMethod, fmt.Sprintf("Error unmarshalling task: %s", err.Error()))
			continue
		}

		// Extract nextExecution attribute value
		nextExecutionAttributeValue, ok := item["nextExecution"]
		if !ok {
			log(callerMethod, "Error: nextExecution not found in item")
			continue
		}
		if nextExecutionAttributeValue == nil {
			log(callerMethod, "Error: nextExecutionAttributeValue is nil")
			continue
		}
		// log(callerMethod, fmt.Sprintf("nextExecution attribute value: %v", nextExecutionAttributeValue))
		// Parse the nextExecution timestamp string into a time.Time object
		nextExecutionTime, err := time.Parse(time.RFC3339, *nextExecutionAttributeValue.S)
		if err != nil {
			log(callerMethod, fmt.Sprintf("Error parsing nextExecution time: %s", err.Error()))
			continue
		}

		// Convert the nextExecution time to Unix timestamp
		nextExecutionUnix := nextExecutionTime.Unix()

		// Create a new job using the retrieved data
		newJob := Job{ID: task.TaskID, Time: nextExecutionUnix}

		// For example, you can add it to a heap using addToHeap(newJob)
		addToHeap(newJob)
	}

	return nil
}

func verifyOwnership(taskID string, userID string) bool {
	return strings.HasPrefix(taskID, userID+"_")
}

func deleteTaskFromDb(taskID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("daria_tasks"), // Specify your DynamoDB tasks table name
		Key: map[string]*dynamodb.AttributeValue{
			"taskId": {
				S: aws.String(taskID),
			},
		},
	}

	_, err := db.svc.DeleteItem(input)
	if err != nil {
		return err
	}
	return nil
}

// getTask retrieves task details from DynamoDB based on the taskId
func getTaskFromDB(taskId string) (*Task, error) {
	startTime := time.Now()
	callerMethod := "getTaskFromDB"
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	// Define input for GetItem operation
	input := &dynamodb.GetItemInput{
		TableName: aws.String("daria_tasks"), // Specify your DynamoDB tasks table name
		Key: map[string]*dynamodb.AttributeValue{
			"taskId": {
				S: aws.String(taskId),
			},
		},
	}

	// Perform the GetItem operation
	result, err := db.svc.GetItem(input)
	if err != nil {
		log(callerMethod, fmt.Sprintf("Error getting task from DynamoDB: %s", err.Error()))
		return nil, err
	}

	// Check if the item exists
	if result.Item == nil {
		log(callerMethod, fmt.Sprintf("Task not found for taskId: %s", taskId))
		return nil, fmt.Errorf("task not found for taskId: %s", taskId)
	}

	// Unmarshal the DynamoDB item into a Task struct
	var task Task
	if err := dynamodbattribute.UnmarshalMap(result.Item, &task); err != nil {
		log(callerMethod, fmt.Sprintf("Error unmarshalling task: %s", err.Error()))
		return nil, err
	}

	return &task, nil
}

func updateTaskInDb(task *Task) {

	// Define input for UpdateItem operation
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("daria_tasks"), // Specify your DynamoDB table name
		Key: map[string]*dynamodb.AttributeValue{
			"taskId": {
				S: aws.String(task.TaskID),
			},
		},
		UpdateExpression: aws.String("SET lastExecution = :le, totalExecutions = :te, nextExecution = :ne"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":le": {
				S: aws.String(task.LastExecution.Format(time.RFC3339)),
			},
			":te": {
				N: aws.String(fmt.Sprintf("%d", task.TotalExecutions)),
			},
			":ne": {
				S: aws.String(task.NextExecution.Format(time.RFC3339)),
			},
		},
	}

	// Perform the UpdateItem operation
	_, err := db.svc.UpdateItem(input)
	if err != nil {
		log("updateTaskInDb", err.Error())
	}
	log("updateTaskInDb", "Updated the task")
}
