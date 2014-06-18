package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"
)

type Task struct {
	description string
	updated     time.Time
}

type TaskList struct {
	tasks []*Task
}

func (t *TaskList) Len() int {
	return len(t.tasks)
}

func (t *TaskList) Less(i, j int) bool {
	return t.tasks[i].updated.After(t.tasks[j].updated)
}

func (t *TaskList) Swap(i, j int) {
	iVal := t.tasks[i]
	t.tasks[i] = t.tasks[j]
	t.tasks[j] = iVal
}

func (t *TaskList) Add(taskDescription string) {
	if t.tasks == nil {
		t.tasks = make([]*Task, 0)
	}
	task := Task{description: taskDescription, updated: time.Now()}
	t.tasks = append(t.tasks, &task)
}

func (t *TaskList) List() []string {
	sort.Sort(t)
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
			task := Task{description: taskDescription, updated: time.Now()}
			t.tasks = append(t.tasks, &task)
		}
	}
	return nil
}

var tasklist *TaskList
var taskFilePath string

func main() {
	tasklist = &(TaskList{})
	var err error
	taskFilePath, err = getTaskFilePath()
	if err != nil {
		fmt.Print(err.Error())
	}
	file, err := os.Open(taskFilePath)
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
	finishFlag := flag.Int("f", -1, "-f 1")
	editFlag := flag.Int("e", -1, "-e 1 newDesc")

	flag.Parse()

	if *finishFlag != -1 {
		tasklist.Finish(*finishFlag)
		err = write(tasklist)
		if err != nil {
			fmt.Print(err.Error())
		}
	} else if *editFlag != -1 {
		if flag.NArg() != 0 {
			tasklist.Edit(*editFlag, strings.Join(flag.Args(), " "))
			err = write(tasklist)
			if err != nil {
				fmt.Print(err.Error())
			}
		} else {
			fmt.Println("No new description given")
		}
	} else if flag.NArg() == 0 {
		for _, task := range tasklist.List() {
			fmt.Println(task)
		}
	} else if flag.NArg() == 1 {
		tasklist.Add(flag.Arg(0))
		err = write(tasklist)
		if err != nil {
			fmt.Print(err.Error())
		}
	} else {
		fmt.Println("NOTHING TO DO?!?")
	}
}

func write(tasklist *TaskList) error {
	marshaledList, _ := tasklist.MarshalText()
	err := ioutil.WriteFile(taskFilePath, marshaledList, 0755)
	if err != nil {
		return err
	}
	return nil
}

func getTaskFilePath() (string, error) {
	tasksFilePath := os.Getenv("T_TASKS_FILE")
	if tasksFilePath == "" {
		user, err := user.Current()
		if err != nil {
			return "", nil
		}
		tasksFilePath = user.HomeDir + "/tasks"
	}
	return tasksFilePath, nil
}
