package client

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Process struct {
	Id      string    `json:"id"`
	App     string    `json:"app"`
	Command string    `json:"command"`
	Host    string    `json:"host"`
	Image   string    `json:"image"`
	Name    string    `json:"name"`
	Ports   []string  `json:"ports"`
	Release string    `json:"release"`
	Cpu     float64   `json:"cpu"`
	Memory  float64   `json:"memory"`
	Started time.Time `json:"started"`
}

type Processes []Process

func (c *Client) GetProcesses(app string, stats bool) (Processes, error) {
	var processes Processes

	err := c.Get(fmt.Sprintf("/apps/%s/processes?stats=%t", app, stats), &processes)

	if err != nil {
		return nil, err
	}

	return processes, nil
}

func (c *Client) GetProcess(app, id string) (*Process, error) {
	var process Process

	err := c.Get(fmt.Sprintf("/apps/%s/processes/%s", app, id), &process)

	if err != nil {
		return nil, err
	}

	return &process, nil
}

func (c *Client) ExecProcessAttached(app, pid, command string, in io.Reader, out io.WriteCloser, height, width int) (int, error) {
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()

	ch := make(chan int)

	go copyWithExit(out, r, ch)

	headers := map[string]string{
		"Command": command,
		"Height":  strconv.Itoa(height),
		"Width":   strconv.Itoa(width),
	}

	err := c.Stream(fmt.Sprintf("/apps/%s/processes/%s/exec", app, pid), headers, in, w)

	w.Close()

	if err != nil {
		return 0, err
	}

	code := <-ch

	return code, nil
}

func (c *Client) RunProcessAttached(app, process, command, release string, height, width int, in io.Reader, out io.WriteCloser) (int, error) {
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()

	ch := make(chan int)

	go copyWithExit(out, r, ch)

	headers := map[string]string{
		"Command": command,
		"Release": release,
		"Height":  strconv.Itoa(height),
		"Width":   strconv.Itoa(width),
	}

	err := c.Stream(fmt.Sprintf("/apps/%s/processes/%s/run", app, process), headers, in, w)
	if err != nil {
		return 0, err
	}

	w.Close()

	code := <-ch

	return code, nil
}

func (c *Client) RunProcessDetached(app, process, command, release string) error {
	var success interface{}

	params := map[string]string{
		"command": command,
		"release": release,
	}

	return c.Post(fmt.Sprintf("/apps/%s/processes/%s/run", app, process), params, &success)
}

func (c *Client) StopProcess(app, id string) (*Process, error) {
	var process Process

	err := c.Delete(fmt.Sprintf("/apps/%s/processes/%s", app, id), &process)

	if err != nil {
		return nil, err
	}

	return &process, nil
}

func copyWithExit(w io.Writer, r io.Reader, ch chan int) {
	buf := make([]byte, 1024)
	code := 1
	state, _ := terminal.MakeRaw(int(os.Stdin.Fd()))

	defer func() {
		if state != nil {
			terminal.Restore(int(os.Stdin.Fd()), state)
		}
		ch <- code
	}()

	for {
		n, err := r.Read(buf)

		if err == io.EOF {
			break
		}

		if err != nil {
			break
		}

		if s := string(buf[0:n]); strings.HasPrefix(s, StatusCodePrefix) {
			code, _ = strconv.Atoi(strings.TrimSpace(s[len(StatusCodePrefix):]))
			return
		}

		_, err = w.Write(buf[0:n])

		if err != nil {
			break
		}
	}
}
