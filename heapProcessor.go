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

	callerMethod := "heapProcessor"
	for {
		isSleeping = false
		queueLock.Lock() // Lock the queue to safely access/modify it
		log(callerMethod, fmt.Sprintf("jobQueue size %d", jobQueue.Len()))
		unlocked := false
		sleepDuration := waitTime * time.Minute
		if len(jobQueue) > 0 {
			job := jobQueue.Peek()
			log(callerMethod, fmt.Sprintf("Next JobId: %s", job.ID))
			currTime := time.Now().Unix()
			if currTime >= job.Time {
				job := jobQueue.Pop().(Job)
				queueLock.Unlock() // Unlock before executing the job
				unlocked = true
				log(callerMethod, fmt.Sprintf("Calling executor for jobId: %s with time %d", job.ID, job.Time))
				//Instead of executing the jobs one at a time, execute them concurrently using go routines. The execution may take time.
				go jobExecutor(job.ID)
				continue
			} else {
				sleepDuration = time.Duration(job.Time-currTime) * time.Second
				log(callerMethod, fmt.Sprintf("Did not execute job: %s with time %s", job.ID, time.Unix(job.Time, 0).Format("2006-01-02 15:04:05")))
			}
		}
		log(callerMethod, fmt.Sprintf("Unlocking heap and sleeping for %s", sleepDuration))
		if !unlocked {
			queueLock.Unlock()
		}
		//If the execution did not happen, that means the top job still has time or there are no jobs left.
		//We make the processor sleep for a while. If there is a new job added during this sleep period, we cancel its sleep and re-runs its logic.
		// Can make this logic super dynamic. For now, the job waits for a fixed period.
		isSleeping = true
		sleeperCtx, sleeperCtxCancel = context.WithCancel(context.Background())
		err := sleep(sleeperCtx, sleepDuration)
		if err != nil {
			log(callerMethod, "Sleep cancelled")
			continue
		}
		log(callerMethod, fmt.Sprintf("Slept for %d seconds", waitTime))
	}
}

func addToHeap(newJob Job) {
	startTime := time.Now()
	callerMethod := "addToHeap"
	log(callerMethod, "Start")
	queueLock.Lock()
	log(callerMethod, "Locked the queue")
	defer func() {
		queueLock.Unlock()
		endLog(callerMethod, startTime)
	}()

	heap.Push(&jobQueue, newJob)
	log(callerMethod, fmt.Sprintf("Added job %s to heap", newJob.ID))

	if isSleeping {
		log(callerMethod, "Cancelling thread sleep due to new job.")
		sleeperCtxCancel()
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
