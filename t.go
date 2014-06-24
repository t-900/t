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

type ByDate []*Task

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].updated.After(b[j].updated)
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (t *TaskList) Add(taskDescription string) {
	if t.tasks == nil {
		t.tasks = make([]*Task, 0)
	}
	task := Task{description: taskDescription, updated: time.Now()}
	t.tasks = append(t.tasks, &task)
}

func (t *TaskList) List() []string {
	tasks := make([]*Task, 0)
	for _, t := range t.tasks {
		tasks = append(tasks, t)
	}
	sort.Sort(ByDate(tasks))
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

func (t *TaskList) Remove(taskId int) error {
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

type parser struct {
	*flag.FlagSet
	edit          int
	finish        int
	remove        int
	list          string
	taskdir       string
	deleteIfEmpty bool
	grep          string
	verbose       bool
	quiet         bool
	done          bool
}

func (p *parser) parseArgs() (parser *parser, args []string) {
	err := p.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}
	return p, p.Args()
}

// Return a parser for the command-line interface.
func build_parser() *parser {
	//usage := "Usage: %prog [-t DIR] [-l LIST] [options] [TEXT]"
	parser := parser{FlagSet: flag.CommandLine}

	parser.IntVar(&parser.edit, "e", -1, "edit TASK to contain TEXT")
	parser.IntVar(&parser.edit, "edit", -1, "edit TASK to contain TEXT")
	parser.IntVar(&parser.finish, "f", -1, "mark TASK as finished")
	parser.IntVar(&parser.finish, "finish", -1, "mark TASK as finished")
	parser.IntVar(&parser.remove, "r", -1, "Remove TASK from list")
	parser.IntVar(&parser.remove, "remove", -1, "Remove TASK from list")

	parser.StringVar(&parser.list, "l", "", "work on LIST")
	parser.StringVar(&parser.list, "list", "", "work on LIST")
	parser.StringVar(&parser.taskdir, "t", "", "work on the lists in DIR")
	parser.StringVar(&parser.taskdir, "task-dir", "", "work on the lists in DIR")
	parser.BoolVar(&parser.deleteIfEmpty, "d", false, "delete the task file if it becomes empty")
	parser.BoolVar(&parser.deleteIfEmpty, "delete-if-empty", false, "delete the task file if it becomes empty")

	parser.StringVar(&parser.grep, "g", "", "print only tasks that contain WORD")
	parser.StringVar(&parser.grep, "grep", "", "print only tasks that contain WORD")
	parser.BoolVar(&parser.verbose, "v", false, "print more detailed output (full task ids, etc)")
	parser.BoolVar(&parser.verbose, "verbose", false, "print more detailed output (full task ids, etc)")
	parser.BoolVar(&parser.quiet, "q", false, "print less detailed output (no task ids, etc)")
	parser.BoolVar(&parser.quiet, "quiet", false, "print less detailed output (no task ids, etc)")
	parser.BoolVar(&parser.done, "done", false, "list done tasks instead of unfinished ones")

	return &parser
}

func main() {
	options, args := build_parser().parseArgs()
	text := strings.TrimSpace(strings.Join(args, " "))
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
	if options.finish != -1 {
		tasklist.Finish(options.finish)
		tasklist.write(options.deleteIfEmpty)
	} else if options.remove != -1 {
		tasklist.Remove(options.remove)
		tasklist.write(options.deleteIfEmpty)
	} else if options.edit != -1 {
		tasklist.Edit(options.edit, text)
		tasklist.write(options.deleteIfEmpty)
	} else if text != "" {
		tasklist.Add(text)
		tasklist.write(options.deleteIfEmpty)
	} else {
		for _, task := range tasklist.List() {
			fmt.Println(task)
		}
	}
}

func (t *TaskList) write(deleteIfEmpty bool) error {
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
