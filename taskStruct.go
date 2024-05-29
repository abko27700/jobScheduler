package main

import "time"

type Task struct {
	TaskID              string                 `json:"taskId"`
	LastExecution       time.Time              `json:"lastExecution"`
	TotalExecutions     int                    `json:"totalExecutions"`
	APIMethod           string                 `json:"apiMethod"`
	APIURL              string                 `json:"apiURL"`
	AvgTimePerExecution float64                `json:"avgTimePerExecution"`
	TimeOutAfter        int                    `json:"timeOutAfter"`
	StartFrom           string                 `json:"startFrom"`
	UserID              string                 `json:"userId"`
	Frequency           int                    `json:"frequency"`
	APIBody             map[string]interface{} `json:"apiBody"`
	NextExecution       time.Time              `json:"nextExecution"`
}

type CreateTaskInput struct {
	APIMethod string                 `json:"apiMethod" binding:"required"`
	APIURL    string                 `json:"apiURL" binding:"required"`
	StartFrom string                 `json:"startFrom" binding:"required"`
	Frequency int                    `json:"frequency" binding:"required"`
	APIBody   map[string]interface{} `json:"apiBody" binding:"required"`
}
