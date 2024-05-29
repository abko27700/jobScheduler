package main

import (
	"container/heap"

	"github.com/gin-gonic/gin"
)

func main() {
	executeBeforeStart()
	r := gin.Default()
	r.Use(apiKeyAuthMiddleware)
	r.POST("/tasks", createTask)
	r.DELETE("/tasks/:taskID", deleteTask)
	r.Run(":8080")
	select {}
}

func executeBeforeStart() {
	clearLogFile("logfile.txt")
	initializeDb()
	jobQueue = make(jobHeap, 0)
	heap.Init(&jobQueue)
	go heapProcessor()
	loadExistingJobs()
}
