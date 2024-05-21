package main

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	jobQueue         jobHeap
	queueLock        sync.Mutex
	sleeperCtx       context.Context
	sleeperCtxCancel context.CancelFunc
	isSleeping       bool
)

// jobExecutor continuously processes jobs from the priority queue.
func heapProcessor() {
	clientId := 1289
	callerMethod := "heapProcessor"
	executionCount := 0
	for {
		isSleeping = false
		queueLock.Lock() // Lock the queue to safely access/modify it
		log(clientId, callerMethod, fmt.Sprintf("jobQueue size %d", jobQueue.Len()))
		unlocked := false
		if len(jobQueue) > 0 {
			job := jobQueue.Peek()
			currTime := time.Now().Unix()
			if currTime >= job.Time {
				job := jobQueue.Pop().(Job)
				queueLock.Unlock() // Unlock before executing the job
				executionCount += 1
				unlocked = true
				log(clientId, callerMethod, fmt.Sprintf("Calling executor for jobId: %d with time %d", job.ID, job.Time))
				//Instead of executing the jobs one at a time, execute them concurrently using go routines. The execution may take time.
				go jobExecutor(job.ID, executionCount)
				continue
			} else {
				log(clientId, callerMethod, fmt.Sprintf("Did not execute job: %d with time %s", job.ID, time.Unix(job.Time, 0).Format("2006-01-02 15:04:05")))
			}
		}
		log(clientId, callerMethod, fmt.Sprintf("Unlocking heap and sleeping for %d seconds", waitTime))
		if !unlocked {
			queueLock.Unlock()
		}
		//If the execution did not happen, that means the top job still has time or there are no jobs left.
		//We make the processor sleep for a while. If there is a new job added during this sleep period, we cancel its sleep and re-runs its logic.
		// Can make this logic super dynamic. For now, the job waits for a fixed period.
		isSleeping = true
		sleeperCtx, sleeperCtxCancel = context.WithCancel(context.Background())
		err := sleep(sleeperCtx, waitTime*time.Second)
		if err != nil {
			log(clientId, callerMethod, "Sleep cancelled")
			continue
		}
		log(clientId, callerMethod, fmt.Sprintf("Slept for %d seconds", waitTime))
	}
}

func addToHeap(newJob Job, clientId int) {
	startTime := time.Now()
	callerMethod := "addToHeap"
	log(clientId, callerMethod, "Start")
	queueLock.Lock()
	log(clientId, callerMethod, "Locked the queue")
	defer func() {
		queueLock.Unlock()
		endLog(clientId, callerMethod, startTime)
	}()

	heap.Push(&jobQueue, newJob)
	log(clientId, callerMethod, fmt.Sprintf("Added job %d to heap", newJob.ID))
	if isSleeping {
		log(clientId, callerMethod, "Cancelling thread sleep due to new job.")
		sleeperCtxCancel()
	}

	//if job is added to heap, wake up the thread/cancel the sleep of heapqueue thread.
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
