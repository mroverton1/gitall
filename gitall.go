package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var MaxOutstanding = runtime.NumCPU()

type Command struct {
	dir string
	args []string
	out  string
}

type CommandProcessor struct {
	input chan *Command
	output chan *Command
	wg *sync.WaitGroup
}

func NewCommandProcessor() *CommandProcessor {
	cp := &CommandProcessor{
		input:make(chan *Command),
		output: make(chan *Command),
	}
	cp.wg = cp.print()
	go cp.work()
	return cp
}

func (cp *CommandProcessor) process(r *Command)  {
	os.Chdir(r.dir)
	cmd := exec.Command("git", r.args...)
	var buf1 bytes.Buffer
	cmd.Stdout = &buf1
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	r.out = buf1.String()
	cp.output <- r
}

func (cp *CommandProcessor) work()  {
	defer close(cp.output)
	var sem = make(chan int, MaxOutstanding)
	var wg sync.WaitGroup
	for cmd := range cp.input {
		sem <- 1
		wg.Add(1)
		go func(cmd *Command) {
			cp.process(cmd)
			<-sem
			wg.Done()
		}(cmd)
	}
	wg.Wait()
}

// the function that handles each file or dir
func (cp *CommandProcessor) visit(path string, info os.FileInfo, err error) error {

	if err != nil {
		fmt.Printf("error 「%v」 at a path 「%q」\n", err, path)
		return err
	}

	if info.IsDir() && strings.HasSuffix(path, "/.git") {

		//fmt.Println("Found .git", path)
		path = path[:len(path)-5]
		cmd := &Command{dir: path, args: os.Args[1:]}
		cp.input <- cmd
	}
	return nil
}

func (cp *CommandProcessor) print() *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for cmd := range cp.output  {
			fmt.Println("--------\n",cmd.dir,"\n")
			fmt.Println(cmd.out)
		}
		wg.Done()
	}()
	return &wg
}

func main() {

	//fmt.Println("Max =", MaxOutstanding)
	cp := NewCommandProcessor()

	// find .git directories
	cwd, _ := os.Getwd()
	err := filepath.Walk(cwd, cp.visit)
	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", cwd, err)
	}
	close(cp.input)
	cp.wg.Wait()
	//fmt.Println("done!")
}
