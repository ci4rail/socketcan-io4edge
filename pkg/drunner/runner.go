package drunner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Runner is a runner object.
type Runner struct {
	id           string
	shallRestart bool
	executable   string
	args         []string
	cmd          *exec.Cmd
}

// New starts the executable with the given arguments and returns a runner object.
// The runner object can be used to stop the executable.
// If the executable terminates, it is restarted again
// stderr and stdout are captured and printed to stdout and stderr with the id as prefix.
func New(id string, executable string, arg ...string) (*Runner, error) {
	r := &Runner{
		id:           id,
		shallRestart: true,
		executable:   executable,
		args:         arg,
	}
	fmt.Printf("starting process for id %s\n", r.id)
	prStdout, prStderr, err := r.startup()
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			r.captureOutput(prStdout, prStderr)

			// wait for executable to terminate
			err = r.cmd.Wait()
			if err != nil {
				fmt.Printf("process for id %s terminated with error: %v\n", r.id, err)
			}
			if !r.shallRestart {
				break
			}
			fmt.Printf("restarting process for id %s\n", r.id)
			prStdout, prStderr, err = r.startup()
			if err != nil {
				fmt.Printf("can't restart process for id %s: %v\n", r.id, err)
				break
			}
		}
	}()
	return r, nil
}

func (r *Runner) startup() (*io.PipeReader, *io.PipeReader, error) {
	r.cmd = exec.Command(r.executable, r.args...)
	//r.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	prStdout, pwStdout := io.Pipe()
	r.cmd.Stdout = pwStdout
	prStderr, pwStderr := io.Pipe()
	r.cmd.Stderr = pwStderr

	err := r.cmd.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("can't start process for id %s: %v", r.id, err)
	}
	return prStdout, prStderr, nil
}

// Stop stops the executable.
// If the executable is not running or can't be stopped, it returns an error.
// Restart is prohibited after Stop.
func (r *Runner) Stop() error {
	r.shallRestart = false

	if r.cmd.Process != nil {
		err := r.cmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("can't kill process for id %s: %v", r.id, err)
		}
	} else {
		return fmt.Errorf("process for id %s is not running", r.id)
	}
	return nil
}

func (r *Runner) captureOutput(prStdout *io.PipeReader, prStderr *io.PipeReader) {
	// stdout
	go func() {
		reader := bufio.NewReader(prStdout)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Printf("%s: %s", r.id, line)
		}
	}()
	// stderr
	go func() {
		reader := bufio.NewReader(prStderr)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Fprintf(os.Stderr, "%s: %s", r.id, line)
		}
	}()
}
