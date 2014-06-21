package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Raised when the path to a task file already exists as a directory.
var InvalidTaskfile string = "Invalid task file: '%s'"

// Raised when trying to use a prefix that could identify multiple tasks.
type AmbiguousPrefix string

// Raised when trying to use a prefix that does not match any tasks.
type UnknownPrefix string

// Return a hash of the given text for use as an id.
// Currently SHA1 hashing is used.  It should be plenty for our purposes.
func hash(text string) string {
	hash := sha1.New()
	io.WriteString(hash, text)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// Parse a taskline (from a task file) and return a task.
//
// A taskline should be in the format:
//
// summary text ... | meta1:meta1_value,meta2:meta2_value,...
//
// The task returned will be a dictionary such as:
//
// { 'id': <hash id>,
// 'text': <summary text>,
// ... other metadata ... }
//
// A taskline can also consist of only summary text, in which case the id
// and other metadata will be generated when the line is read.  This is
// supported to enable editing of the taskfile with a simple text editor.
func task_from_taskline(taskline string) map[string]string {
	task := make(map[string]string)
	if strings.HasPrefix(strings.TrimSpace(taskline), "#") {
		return nil
	} else if strings.Contains(taskline, "|") {
		sepIndex := strings.LastIndex(taskline, "|")
		text := taskline[:sepIndex]
		meta := taskline[sepIndex+1:]
		task["text"] = strings.TrimSpace(text)
		for _, piece := range strings.Split(strings.TrimSpace(meta), ",") {
			metaSplit := strings.Split(piece, ":")
			var label, data string
			for i, m := range metaSplit {
				if i == 0 {
					label = strings.TrimSpace(m)
				} else if i == 1 {
					data = strings.TrimSpace(m)
				}
			}
			task[label] = data
		}
	} else {
		text := strings.TrimSpace(taskline)
		task["id"] = hash(text)
		task["text"] = text
	}
	return task
}

// Parse a list of tasks into tasklines suitable for writing.
func tasklines_from_tasks(tasks map[string]map[string]string) []string {
	tasklines := make([]string, 0)

	for _, task := range tasks {
		meta := make([][]string, 0)
		for k, v := range task {
			if k != "text" {
				meta = append(meta, []string{k, v})
			}
		}
		meta_str := ""
		for _, m := range meta {
			if len(meta_str) > 0 {
				meta_str = meta_str + ", "
			}
			meta_str = meta_str + fmt.Sprintf("%s:%s", m[0], m[1])
		}
		tasklines = append(tasklines, fmt.Sprintf("%s | %s\n", task["text"], meta_str))
	}

	return tasklines
}

// Return a mapping of ids to prefixes in O(n) time.
//
// Each prefix will be the shortest possible substring of the ID that
// can uniquely identify it among the given group of IDs.
//
// If an ID of one task is entirely a substring of another task's ID, the
// entire ID will be the prefix.
func prefixes(tasks map[string]map[string]string) map[string]string {
	ids := make([]string, 0)
	for _, task := range tasks {
		ids = append(ids, task["id"])
	}
	ps := make(map[string]string)
	var prefix string
	for _, id := range ids {
		id_len := len(id)
		i := 1
		for ; i < id_len+1; i++ {
			// identifies an empty prefix slot, or a singular collision
			prefix = id[:i]
			if ps_prefix, in_ps := ps[prefix]; !in_ps || (ps_prefix != prefix) {
				break
			}
		}
		if other_id, in_ps := ps[prefix]; in_ps {
			// if there is a collision
			for j := i; j < id_len+1; j++ {
				if other_id[:j] == id[:j] {
					ps[id[:j]] = ""
				} else {
					ps[other_id[:j]] = other_id
					ps[id[:j]] = id
					break
				}
			}
			ps[other_id[:id_len+1]] = other_id
			ps[id] = id
		} else {
			// no collision, can safely add
			ps[prefix] = id
		}
	}
	ps_swapped := make(map[string]string)
	for k, v := range ps {
		ps_swapped[v] = k
	}
	delete(ps_swapped, "")
	return ps_swapped
}

// A set of tasks, both finished and unfinished, for a given list.
//
// The list's files are read from disk when the TaskDict is initialized. They
// can be written back out to disk with the write() function.
//
type TaskDict struct {
	tasks, done   map[string]map[string]string
	name, taskdir string
}

func (t *TaskDict) tasksForKind(kind string) map[string]map[string]string {
	if kind == "tasks" {
		return t.tasks
	} else if kind == "done" {
		return t.done
	} else {
		panic("No such kind: '" + kind + "'")
	}
}

// Initialize by reading the task files, if they exist.
func NewTaskDict(taskdir string, name string) *TaskDict {
	taskDict := TaskDict{}
	taskDict.tasks = make(map[string]map[string]string)
	taskDict.done = make(map[string]map[string]string)
	if name == "" {
		name = "tasks"
	}
	taskDict.name = name
	if taskdir == "" {
		taskdir = "."
	}
	taskDict.taskdir = taskdir
	filemap := map[string]string{"tasks": taskDict.name, "done": fmt.Sprintf(".%s.done", taskDict.name)}
	for kind, filename := range filemap {
		path := path.Join(taskDict.taskdir, filename)
		fileinfo, err := os.Stat(path)
		if os.IsNotExist(err) {
			break
		}
		if fileinfo.IsDir() {
			panic(fmt.Errorf(InvalidTaskfile, path))
		}
		file, err := os.Open(path)
		defer file.Close()
		if err != nil {
			panic(err)
		}
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		fileLines := string(fileBytes)
		for _, line := range strings.Split(fileLines, "\n") {
			if line != "" {
				task := task_from_taskline(strings.TrimSpace(line))
				if task != nil {
					tasks := taskDict.tasksForKind(kind)
					tasks[task["id"]] = task
				}
			}
		}
	}
	return &taskDict
}

// Return the unfinished task with the given prefix.
//
// If more than one task matches the prefix an AmbiguousPrefix exception
// will be raised, unless the prefix is the entire ID of one task.
//
// If no tasks match the prefix an UnknownPrefix exception will be raised.
func (t *TaskDict) getItem(prefix string) map[string]string {
	matched := make([]string, 0)
	for k, _ := range t.tasks {
		if strings.HasPrefix(k, prefix) {
			matched = append(matched, k)
		}
	}
	if len(matched) == 1 {
		return t.tasks[matched[0]]
	} else if len(matched) == 0 {
		panic(UnknownPrefix(prefix))
	} else {
		matched = make([]string, 0)
		for k, _ := range t.tasks {
			if k == prefix {
				matched = append(matched, k)
			}
		}
		if len(matched) == 1 {
			return t.tasks[matched[0]]
		} else {
			panic(AmbiguousPrefix(prefix))
		}
	}
}

// Add a new, unfinished task with the given summary text.
func (t *TaskDict) add_task(text string) {
	task_id := hash(text)
	t.tasks[task_id] = map[string]string{"id": task_id, "text": text}
}

// Edit the task with the given prefix.
//
// If more than one task matches the prefix an AmbiguousPrefix exception
// will be raised, unless the prefix is the entire ID of one task.
//
// If no tasks match the prefix an UnknownPrefix exception will be raised.
func (t *TaskDict) edit_task(prefix, text string) {
	task := t.getItem(prefix)
	if strings.HasPrefix(text, "s/") || strings.HasPrefix(text, "/") {
		text := strings.TrimSpace(regexp.MustCompile("^s?/").ReplaceAllLiteralString(text, ""))
		i := strings.Index(text, "/")
		find := text[:i]
		repl := text[i+1:]
		text = strings.Replace(task["text"], find, repl, 0)
	}
	task["text"] = text
}

// Mark the task with the given prefix as finished.
//
// If more than one task matches the prefix an AmbiguousPrefix exception
// will be raised, if no tasks match it an UnknownPrefix exception will
// be raised.
func (t *TaskDict) finish_task(prefix string) {
	id := t.getItem(prefix)["id"]
	if task, ok := t.tasks[id]; ok {
		delete(t.tasks, id)
		t.done[task["id"]] = task
	}
}

// Remove the task from tasks list.
//
// If more than one task matches the prefix an AmbiguousPrefix exception
// will be raised, if no tasks match it an UnknownPrefix exception will
// be raised.
func (t *TaskDict) remove_task(prefix string) {
	delete(t.tasks, t.getItem(prefix)["id"])
}

// Print out a nicely formatted list of unfinished tasks.
func (t *TaskDict) print_list(kind string, verbose bool, quiet bool, grep string) {
	if kind == "" {
		kind = "tasks"
	}
	tasks := t.tasksForKind(kind)
	label := "prefix"
	if verbose {
		label = "id"
	}

	if !verbose {
		for task_id, prefix := range prefixes(tasks) {
			tasks[task_id]["prefix"] = prefix
		}
	}
	var keys []string
	for k := range tasks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	plen := 0
	if len(tasks) > 0 {
		var keyLength []int
		for _, task := range tasks {
			keyLength = append(keyLength, len(task[label]))
		}
		sort.Ints(keyLength)
		plen = keyLength[len(keyLength)-1]
	}
	for _, sortedKey := range keys {
		task := tasks[sortedKey]
		if strings.Contains(strings.ToLower(task["text"]), strings.ToLower(grep)) {
			p := ""
			if !quiet {
				template := fmt.Sprintf("%%-%ds - ", plen)
				p = fmt.Sprintf(template, task[label])
			}
			fmt.Println(p + task["text"])
		}
	}
}

// Flush the finished and unfinished tasks to the files on disk.
func (t *TaskDict) write(delete_if_empty bool) {
	filemap := map[string]string{"tasks": t.name, "done": fmt.Sprintf(".%s.done", t.name)}
	for kind, filename := range filemap {
		path := path.Join(t.taskdir, filename)
		fileInfo, err := os.Stat(path)
		if !os.IsNotExist(err) {
			if fileInfo.IsDir() {
				panic(fmt.Errorf(InvalidTaskfile, path))
			}
		}
		tasks := t.tasksForKind(kind)
		var ids []string
		for _, v := range tasks {
			ids = append(ids, v["id"])
		}
		sort.Strings(ids)
		if len(tasks) > 0 || !delete_if_empty {
			tasklines := tasklines_from_tasks(tasks)
			data := []byte(strings.Join(tasklines, "\n"))
			err = ioutil.WriteFile(path, data, 0755)
			if err != nil {
				if !os.IsNotExist(err) {
					panic(err)
				}
			}
		} else if len(tasks) == 0 && !fileInfo.IsDir() {
			os.Remove(path)
		}
	}
}

type parser struct {
	*flag.FlagSet
	edit          string
	finish        string
	remove        string
	list          string
	taskdir       string
	deleteIfEmpty bool
	grep          string
	verbose       bool
	quiet         bool
	done          bool
}

func (p *parser) parse_args() (parser *parser, args []string) {
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

	parser.StringVar(&parser.edit, "e", "", "edit TASK to contain TEXT")
	parser.StringVar(&parser.edit, "edit", "", "edit TASK to contain TEXT")
	parser.StringVar(&parser.finish, "f", "", "mark TASK as finished")
	parser.StringVar(&parser.finish, "finish", "", "mark TASK as finished")
	parser.StringVar(&parser.remove, "r", "", "Remove TASK from list")
	parser.StringVar(&parser.remove, "remove", "", "Remove TASK from list")

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

// Run the command-line interface.
func main() {
	options, args := build_parser().parse_args()

	td := NewTaskDict(options.taskdir, options.list)
	text := strings.TrimSpace(strings.Join(args, " "))

	defer func() {
		if e := recover(); e != nil {
			if err, ok := e.(AmbiguousPrefix); ok {
				fmt.Fprintf(os.Stderr, `The ID "%s" matches more than one task.`+"\n", err)
			} else if err, ok := e.(UnknownPrefix); ok {
				fmt.Fprintf(os.Stderr, `The ID "%s" does not match any task.`+"\n", err)
			} else {
				fmt.Fprint(os.Stderr, e)
			}
		}
	}()

	if options.finish != "" {
		td.finish_task(options.finish)
		td.write(options.deleteIfEmpty)
	} else if options.remove != "" {
		td.remove_task(options.remove)
		td.write(options.deleteIfEmpty)
	} else if options.edit != "" {
		td.edit_task(options.edit, text)
		td.write(options.deleteIfEmpty)
	} else if text != "" {
		td.add_task(text)
		td.write(options.deleteIfEmpty)
	} else {
		kind := "tasks"
		if options.done {
			kind = "done"
		}
		td.print_list(kind, options.verbose, options.quiet, options.grep)
	}
}

//
//
//if __name__ == '__main__':
//    _main()
