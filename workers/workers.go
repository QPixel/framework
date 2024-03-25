package workers

// workers.go
// This package contains the necessary code to schedule reoccurring events
// Workers can manipulate different parts of the bot and are responsible for
// Mutes, TempBans, Presence updates, and other required things
// Commands can also register workers with the manager

// todo clean up the documentation

// WORKERS RUN MULTIPLE TIMES WHILE THE BOT IS RUNNING
// JOBS ARE THE ACTUAL GOCRON VERSION OF the WORKER

import (
	"time"

	"github.com/go-co-op/gocron"
	tlog "github.com/ubergeek77/tinylog"
)

var wlog = tlog.NewTaggedLogger("WorkerManager", tlog.NewColor("38;5;111"))

// WorkerManager is an easy way to manage workers.
type WorkerManager struct {
	Scheduler *gocron.Scheduler
	Workers   map[string]Worker
	Jobs      []*gocron.Job
	IsRunning bool
}

// Worker
// Describes a worker.
type Worker struct {
	Duration   string
	WorkerFunc func()
}

func InitializeManager(loc *time.Location) *WorkerManager {
	wrk := &WorkerManager{
		Scheduler: gocron.NewScheduler(loc),
		Workers:   make(map[string]Worker),
		IsRunning: false,
	}
	wrk.Scheduler.TagsUnique()
	return wrk
}

// Start
// Will start all the workers via the scheduler.
func (m *WorkerManager) Start() {
	m.Scheduler.StartAsync()
	m.IsRunning = true
}

// StopWorkers
// Will stop all the workers via the scheduler.
func (m *WorkerManager) StopWorkers() {
	m.Scheduler.StopBlockingChan()
	m.IsRunning = false
}

// AddWorker
// Adds a worker to the internal worker map.
func (m *WorkerManager) AddWorker(tag string, worker Worker) {
	m.Workers[tag] = worker
}

// AddWorkers
// registers all the workers to the scheduler.
func (m *WorkerManager) AddWorkers() {
	for tag, worker := range m.Workers {
		job, err := m.Scheduler.Cron(worker.Duration).Tag(tag).Do(worker.WorkerFunc)
		if err != nil {
			wlog.Errorf("Unable to register worker %s", tag)
			wlog.Fatal(err.Error())
		}
		m.Jobs = append(m.Jobs, job)
	}
}

// RemoveWorker
// Removes a worker from the scheduler.
func (m *WorkerManager) RemoveWorker() {

}

// AddWorkerOnce
// Easy way to add a single job to the scheduler.
func (m *WorkerManager) AddWorkerOnce() {

}
