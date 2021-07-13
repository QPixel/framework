package framework

import (
	"sync"
	"time"
)

// workers.go
// This file contains everything for adding and managing workers

// workerLock
// A map that stores mutexes for the background workers
// These will be used to determine when the workers have exited gracefully
// If a worker is still locked, then it has not exited
var workerLock = make(map[int]*sync.Mutex)

// workers
// The list of workers that are to be pre-registered before the bot starts, then all executed in the background
var workers []func()

// continueLoop
// This boolean will be changed to false when the bot is trying to shut down
// All the background workers are looping on this being true, meaning they will stop when it is false
var continueLoop = true

// AddWorker
// Given a function that is passed through, append it to the list of worker functions
func AddWorker(worker func()) {
	workers = append(workers, worker)
}

// startWorkers
// Go through the list of workers than have been added to the list, and execute them all in the background
func startWorkers() {
	// Iterate over all the workers
	for i, worker := range workers {
		// Create a mutex for this worker
		workerLock[i] = &sync.Mutex{}

		// Start a goroutine for this worker, which starts it in the background
		go func(worker func(), i int) {
			// Lock the worker; this will be used in graceful termination
			workerLock[i].Lock()

			// Run the worker once per second, forever, until a TERM signal breaks this loop
			for continueLoop {
				worker()
				time.Sleep(time.Second)
			}

			// The loop has stopped. Unlock the worker
			workerLock[i].Unlock()
		}(worker, i)
	}
}
