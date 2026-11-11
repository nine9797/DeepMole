package runner

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Runner struct {
	Exe string
	Dir string

	mu   sync.Mutex
	cmd  *exec.Cmd
	out  bytes.Buffer
	done chan struct{}
	err  error
}

var (
	jobOnce sync.Once
	job     windows.Handle
)

func jobObject() windows.Handle {
	jobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
			windows.CloseHandle(h)
			return
		}
		job = h
	})
	return job
}

func attachToJob(pid int) {
	j := jobObject()
	if j == 0 {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	_ = windows.AssignProcessToJobObject(j, h)
}

func hidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

func KillStray() {
	cmd := exec.Command("taskkill", "/F", "/IM", "winws.exe")
	hidden(cmd)
	_ = cmd.Run()
}

func (r *Runner) Start(args []string) error {
	r.Stop()
	KillStray()

	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command(r.Exe, args...)
	cmd.Dir = r.Dir
	hidden(cmd)
	r.out.Reset()
	cmd.Stdout = &r.out
	cmd.Stderr = &r.out
	if err := cmd.Start(); err != nil {
		return err
	}
	attachToJob(cmd.Process.Pid)
	r.cmd = cmd
	r.done = make(chan struct{})
	done := r.done
	go func() {
		r.err = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		msg := strings.TrimSpace(r.out.String())
		if msg == "" {
			msg = "winws exited immediately"
		}
		r.cmd = nil
		return errors.New(msg)
	case <-time.After(600 * time.Millisecond):
		return nil
	}
}

func (r *Runner) Stop() {
	r.mu.Lock()
	cmd, done := r.cmd, r.done
	r.cmd = nil
	r.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil {
		return false
	}
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

func (r *Runner) Output() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.out.String()
}
