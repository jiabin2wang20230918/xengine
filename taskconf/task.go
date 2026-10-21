package taskconf

type Task interface {
	//NewTask() *Task
	StartTask() bool
	StopTask() bool
	AddTask() error
	SyncTask(newTask *taskData) error
	DeleteTask() error
}
