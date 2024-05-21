package main

import (
	"container/heap"

	"github.com/gin-gonic/gin"
)

func main() {
	clearLogFile("logfile.txt")
	jobQueue = make(jobHeap, 0)
	heap.Init(&jobQueue)
	go heapProcessor()
	loadExistingJobs()

	r := gin.Default()
	r.POST("/tasks", createTask)
	r.Run(":8080")

	// jobExecutor()
	select {}
}
