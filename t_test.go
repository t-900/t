package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestCliAddTask(t *testing.T) {
	withCliSetup(t, func() {
		cmd := exec.Command("go", "run", "t.go", "foo")
		err := cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		listCmd := exec.Command("go", "run", "t.go")
		out, err := listCmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		outString := string(out)
		expected := "0 - foo\n"
		if outString != expected {
			t.Fatalf("Expected output to be '%s', got '%s'", expected, outString)
		}
	})
}

func TestCliFinishTask(t *testing.T) {
	withCliSetup(t, func() {
		cmd := exec.Command("go", "run", "t.go", "foo")
		err := cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		finishCmd := exec.Command("go", "run", "t.go", "-f", "0")
		err = finishCmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		listCmd := exec.Command("go", "run", "t.go")
		out, err := listCmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		outString := string(out)
		if outString != "" {
			t.Fatalf("Expected output to be '', got '%s'", outString)
		}
	})
}

func TestCliEditTask(t *testing.T) {
	withCliSetup(t, func() {
		cmd := exec.Command("go", "run", "t.go", "foo")
		err := cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		editCmd := exec.Command("go", "run", "t.go", "-e", "0", "bar")
		err = editCmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		listCmd := exec.Command("go", "run", "t.go")
		out, err := listCmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		outString := string(out)
		expected := "0 - bar\n"
		if outString != expected {
			t.Fatalf("Expected output to be '%s', got '%s'", expected, outString)
		}
	})
}

func withCliSetup(t *testing.T, testFunc func()) {
	origTaskFilePath := os.Getenv("T_TASKS_FILE")
	err := os.Setenv("T_TASKS_FILE", "/tmp/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Remove("/tmp/tasks")
		os.Setenv("T_TASKS_FILE", origTaskFilePath)
	}()
	testFunc()
}

func TestAddTask(t *testing.T) {
	tasklist := TaskList{}
	tasklist.Add("foo")
	if len(tasklist.tasks) != 1 {
		t.Fatalf("Expected list to have one element, got %d", len(tasklist.tasks))
	}

	actualTaskDescription := tasklist.tasks[0].description
	if actualTaskDescription != "foo" {
		t.Fatalf("expected tasklist to contain 'foo', got '%v'", actualTaskDescription)
	}
}

func TestListTasks(t *testing.T) {
	tasklist := TaskList{}
	tasks := tasklist.List()

	if len(tasks) != 0 {
		t.Fatalf("Expected tasklist to contain no element, got %d", len(tasks))
	}

	tasklist.Add("Foo")
	tasks = tasklist.List()

	if len(tasks) != 1 {
		t.Fatalf("Expected tasklist to have one element, got %d", len(tasks))
	}
}

func TestFinishTask(t *testing.T) {
	tasklist := TaskList{}
	tasklist.Add("foo")

	if len(tasklist.tasks) != 1 {
		t.Fatalf("Expected tasklist to contain one element, got %d", len(tasklist.tasks))
	}

	tasklist.Finish(0)

	if len(tasklist.tasks) != 0 {
		t.Fatalf("Expected tasklist to contain no element, got %d", len(tasklist.tasks))
	}
}

func TestEditTask(t *testing.T) {
	tasklist := TaskList{}
	tasklist.Add("foo")

	if len(tasklist.tasks) != 1 {
		t.Fatalf("Expected tasklist to contain one element, got %d", len(tasklist.tasks))
	}
	actualTaskDescription := tasklist.tasks[0].description
	if actualTaskDescription != "foo" {
		t.Fatalf("expected tasklist to contain 'foo', got '%v'", actualTaskDescription)
	}

	tasklist.Edit(0, "bar")

	actualTaskDescription = tasklist.tasks[0].description
	if actualTaskDescription != "bar" {
		t.Fatalf("expected tasklist to contain 'bar', got '%v'", actualTaskDescription)
	}
}
