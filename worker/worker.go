package worker

import (
	"errors"
	"fmt"
	"log"
	"orchestrator/task"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) CollectStats() {}

func (w *Worker) RunTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("the queue has no tasks")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskPersisted, ok := w.Db[taskQueued.ID]
	if !ok {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued

	}
	var result task.DockerResult
	defer func() {
		log.Println("returning result: ", result)
	}()
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			fmt.Println("scheduled: ", taskQueued)
			result = w.StartTask(taskQueued)
		case task.Completed:
			fmt.Println("completed: ", taskQueued)
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("We should not get here")
		}
	} else {
		err := fmt.Errorf("Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}
	return result

}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		log.Printf("Error running task %v: %v", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}

	t.ContainerId = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t
	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Stop(d.ContainerId)
	if result.Error != nil {
		log.Printf("error stopping container %s: %s", d.ContainerId, result.Error)
	}
	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t
	return result
}
