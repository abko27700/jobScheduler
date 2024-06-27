package main

import (
	"container/heap"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
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
	log("executeBeforeStart", "Starting executeBeforeStart")
	initializeDb()
	jobQueue = make(jobHeap, 0)
	heap.Init(&jobQueue)
	go heapProcessor()
	loadExistingJobs()
}
