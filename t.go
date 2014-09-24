package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
)

type Task struct {
	description string
}

type TaskList struct {
	tasks []*Task
}

func (t *TaskList) Add(taskDescription string) {
	if t.tasks == nil {
		t.tasks = make([]*Task, 0)
	}
	task := Task{description: taskDescription}
	t.tasks = append(t.tasks, &task)
}

func (t *TaskList) List() []string {
	tasks := make([]*Task, 0)
	for _, t := range t.tasks {
		tasks = append(tasks, t)
	}
	list := make([]string, 0)
	for i, task := range t.tasks {
		list = append(list, fmt.Sprintf("%d - %s", i, task.description))
	}
	return list
}

func (t *TaskList) Finish(taskId int) error {
	if t.tasks == nil {
		return errors.New("No tasks found")
	}
	if len(t.tasks) <= taskId {
		return errors.New("No task for id found")
	}
	newTasks := make([]*Task, 0)
	for i, task := range t.tasks {
		if i != taskId {
			newTasks = append(newTasks, task)
		}
	}
	t.tasks = newTasks
	return nil
}

func (t *TaskList) Edit(taskId int, newDescription string) error {
	if t.tasks == nil {
		return errors.New("No tasks found")
	}
	if len(t.tasks) <= taskId {
		return errors.New("No task for id found")
	}
	t.tasks[taskId].description = newDescription
	return nil
}

func (t *TaskList) MarshalText() ([]byte, error) {
	list := make([]string, 0)
	for _, task := range t.tasks {
		list = append(list, task.description)
	}
	return []byte(strings.Join(list, "\n")), nil
}

func (t *TaskList) UnmarshalText(text []byte) error {
	in := string(text)
	list := strings.Split(in, "\n")

	t.tasks = make([]*Task, 0)
	for _, taskDescription := range list {
		if taskDescription != "" {
			task := Task{description: taskDescription}
			t.tasks = append(t.tasks, &task)
		}
	}
	return nil
}

var tasklist *TaskList
var taskFilePath string

func main() {
	var (
		editTask   = flag.Int("e", -1, "edit the tasklist")
		finishTask = flag.Int("f", -1, "finish task #")
	)

	flag.Parse()

	tasklist = &(TaskList{})
	taskFilePath = getTaskFilePath()

	file, err := os.Open(taskFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Print(err)
		}
	}
	defer file.Close()
	if file != nil {
		taskBytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Print(err.Error())
		}
		err = tasklist.UnmarshalText(taskBytes)
		if err != nil {
			fmt.Print(err.Error())
		}
	}

	text := strings.Join(flag.Args(), " ")
	if *editTask != -1 {
		tasklist.Edit(*editTask, text)
		tasklist.write(true)
	} else if *finishTask != -1 {
		tasklist.Finish(*finishTask)
		tasklist.write(true)
	} else {
		if len(flag.Args()) > 0 {
			tasklist.Add(text)
			tasklist.write(true)
		} else {
			for _, task := range tasklist.List() {
				fmt.Println(task)
			}
		}
	}
}

func (t *TaskList) write(deleteIfEmpty bool) error {
	marshaledList, _ := tasklist.MarshalText()
	err := ioutil.WriteFile(taskFilePath, marshaledList, 0644)
	if err != nil {
		return err
	}
	return nil
}

func getTaskFilePath() string {
	tasksFilePath := os.Getenv("T_TASKS_FILE")
	if tasksFilePath == "" {
		user, _ := user.Current()
		tasksFilePath = user.HomeDir + "/tasks"
	}
	return tasksFilePath
}
