package taskconf

type task interface {
	StartTask() bool
	StopTask() bool
	AddTask() error
	SyncTask(newTask *taskData) error
	DeleteTask() error
}
