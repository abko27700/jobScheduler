package main

type jobHeap []Job

// Implementing heap.Interface methods for jobHeap
func (h jobHeap) Len() int           { return len(h) }
func (h jobHeap) Less(i, j int) bool { return h[i].Time > h[j].Time }
func (h jobHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h jobHeap) Peek() Job {
	if len(h) == 0 {
		return Job{} // Return an empty Job if the heap is empty
	}
	return h[0]
}

func (h *jobHeap) Push(x interface{}) {
	*h = append(*h, x.(Job))
}

func (h *jobHeap) Pop() interface{} {
	if h.Len() == 0 {
		return nil
	}

	jobQ := *h
	qSize := len(jobQ)
	job := jobQ[0]
	*h = jobQ[1:qSize]

	return job
}
