package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials("your_access_key_id", "your_secret_access_key", ""),
	}))

	return &DynamoDBClient{
		svc: dynamodb.New(sess),
	}
}

// InitializeDb initializes the DynamoDB client
func initializeDb() {
	db = NewDynamoDBClient("us-east-1") // Specify your preferred AWS region
}

// ValidateUser checks if the user exists in the DynamoDB table
func (db *DynamoDBClient) ValidateUser(userID string) (bool, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String("YourUserTableName"), // Specify your DynamoDB user table name
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
	input := &dynamodb.GetItemInput{
		TableName: aws.String("daria_jobs_apiKeys"), // Specify your DynamoDB API keys table name
		Key: map[string]*dynamodb.AttributeValue{
			"APIKey": {
				S: aws.String(apiKey),
			},
		},
	}

	result, err := d.svc.GetItem(input)
	if err != nil {
		return "", err
	}

	if result.Item == nil {
		return "", nil
	}

	userID := result.Item["UserID"].S
	return *userID, nil
}

func apiKeyAuthMiddleware(c *gin.Context) {
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

	// Store the userID in the context for further use
	c.Set("userID", userID)

	c.Next() // Pass control to the next middleware/handler
}

func validateUser(c *gin.Context) bool {
	// Retrieve the userID from the context
	userID := c.GetString("userID")

	// Query the daria_users table to retrieve jobLimit and jobCount for the userID
	jobLimit, jobCount, err := getUserJobLimits(userID)
	if err != nil {
		// Handle error
		// For example, return false and log the error
		fmt.Println("Error:", err)
		return false
	}

	// Compare jobCount with jobLimit
	if jobCount >= jobLimit {
		// Return an error message if jobCount exceeds jobLimit
		errorMessage := fmt.Sprintf("Maximum job limit (%d) has been reached", jobLimit)
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMessage})
		return false
	}

	// User is valid
	return true
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
