package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DeletedTask struct {
	TaskID string
}

var deletedTasks sync.Map

func deleteTask(c *gin.Context) {
	callerMethod := "deleteTask"
	startTime := time.Now()
	log(callerMethod, "Start")
	defer func() {
		endLog(callerMethod, startTime)
	}()

	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// Verify ownership of the task
	userID := c.GetString("userId")

	if !verifyOwnership(taskID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete this task"})
		return
	}

	err := deleteTaskFromDb(taskID)

	if err != nil {
		log(callerMethod, fmt.Sprintf("Error deleting task: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the task"})
		return
	}

	addDeletedTask(taskID)

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func addDeletedTask(taskID string) {
	deletedTasks.Store(taskID, DeletedTask{
		TaskID: taskID,
	})
}

func isTaskDeleted(taskID string) bool {
	_, ok := deletedTasks.Load(taskID)
	return ok
}
