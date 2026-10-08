package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

type managedProcess struct {
	cmd       *exec.Cmd
	startedAt time.Time
	lastError string
}

type RuntimeStatus struct {
	Mode      string `json:"mode"`
	Managed   bool   `json:"managed"`
	State     string `json:"state"`
	PID       int    `json:"pid,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

func (m *Manager) commandFor(manifest Manifest) (*Command, error) {
	if manifest.Server.Runtime == nil {
		return nil, errors.New("this plugin is an external service and has no managed runtime")
	}
	var command *Command
	switch goruntime.GOOS {
	case "windows":
		command = manifest.Server.Runtime.Windows
	case "linux":
		command = manifest.Server.Runtime.Linux
	case "darwin":
		command = manifest.Server.Runtime.Darwin
	}
	if command == nil {
		return nil, fmt.Errorf("plugin has no runtime command for %s", goruntime.GOOS)
	}
	return command, nil
}

func (m *Manager) Start(ctx context.Context, id string) error {
	plugin, err := m.Get(id)
	if err != nil {
		return err
	}
	if !plugin.Enabled {
		return errors.New("plugin is disabled")
	}
	if plugin.Manifest.Server.StartMode == "external" {
		return errors.New("external plugin services must be started by their own application")
	}

	m.runtimeMu.Lock()
	if process := m.processes[id]; process != nil && process.cmd.Process != nil && process.cmd.ProcessState == nil {
		m.runtimeMu.Unlock()
		return nil
	}
	command, err := m.commandFor(plugin.Manifest)
	if err != nil {
		m.runtimeMu.Unlock()
		return err
	}
	cmd := exec.Command(command.Executable, command.Args...)
	cmd.Dir = command.WorkingDirectory
	logDir := filepath.Join("data", "logs", "plugins")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		m.runtimeMu.Unlock()
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, id+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		m.runtimeMu.Unlock()
		return err
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		m.runtimeMu.Unlock()
		m.audit(id, "runtime-start", false, 0, err.Error())
		return fmt.Errorf("start plugin runtime: %w", err)
	}
	process := &managedProcess{cmd: cmd, startedAt: time.Now()}
	m.processes[id] = process
	delete(m.lastRuntimeErrors, id)
	m.runtimeMu.Unlock()
	m.audit(id, "runtime-start", true, 0, fmt.Sprintf("process %d started", cmd.Process.Pid))
	go func() {
		waitErr := cmd.Wait()
		_ = logFile.Close()
		m.runtimeMu.Lock()
		if current := m.processes[id]; current == process {
			if waitErr != nil {
				current.lastError = waitErr.Error()
				m.lastRuntimeErrors[id] = waitErr.Error()
			}
			delete(m.processes, id)
		}
		m.runtimeMu.Unlock()
		m.audit(id, "runtime-exit", waitErr == nil, 0, errorMessage(waitErr))
	}()

	timeout := 30 * time.Second
	if value := plugin.Manifest.Server.Runtime.StartupTimeoutSeconds; value > 0 {
		timeout = time.Duration(value) * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.runtimeMu.Lock()
		_, stillRunning := m.processes[id]
		m.runtimeMu.Unlock()
		if !stillRunning {
			return fmt.Errorf("plugin process exited before becoming ready: %s", m.runtimeFailureDetail(id))
		}
		_, status, healthErr := m.request(ctx, plugin.Manifest, "GET", plugin.Manifest.Server.HealthEndpoint, nil, 2*time.Second)
		if healthErr == nil && status >= 200 && status < 300 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	_ = m.Stop(id)
	return fmt.Errorf("plugin process started but its health endpoint did not become ready: %s", m.runtimeFailureDetail(id))
}

func (m *Manager) Stop(id string) error {
	m.runtimeMu.Lock()
	process := m.processes[id]
	m.runtimeMu.Unlock()
	if process == nil || process.cmd.Process == nil {
		return nil
	}
	err := process.cmd.Process.Kill()
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	m.audit(id, "runtime-stop", true, 0, "managed process stopped")
	return nil
}

func (m *Manager) Restart(ctx context.Context, id string) error {
	if err := m.Stop(id); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return m.Start(ctx, id)
}

func (m *Manager) RuntimeStatus(id string) (RuntimeStatus, error) {
	plugin, err := m.Get(id)
	if err != nil {
		return RuntimeStatus{}, err
	}
	status := RuntimeStatus{Mode: plugin.Manifest.Server.StartMode, Managed: plugin.Manifest.Server.StartMode != "external", State: "stopped"}
	m.runtimeMu.Lock()
	defer m.runtimeMu.Unlock()
	if process := m.processes[id]; process != nil && process.cmd.Process != nil {
		status.State = "running"
		status.PID = process.cmd.Process.Pid
		status.StartedAt = process.startedAt.Format(time.RFC3339)
		status.LastError = process.lastError
	} else if detail := m.lastRuntimeErrors[id]; detail != "" {
		status.State = "crashed"
		status.LastError = detail
	}
	return status, nil
}

func (m *Manager) runtimeFailureDetail(id string) string {
	path := filepath.Join("data", "logs", "plugins", id+".log")
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if len(data) > 3000 {
			data = data[len(data)-3000:]
		}
		if text := strings.TrimSpace(string(data)); text != "" {
			return text
		}
	}
	m.runtimeMu.Lock()
	defer m.runtimeMu.Unlock()
	if detail := m.lastRuntimeErrors[id]; detail != "" {
		return detail
	}
	return "no diagnostic output was produced"
}

func (m *Manager) startAutomaticPlugins() {
	time.Sleep(500 * time.Millisecond)
	items, err := m.List()
	if err != nil {
		return
	}
	for _, item := range items {
		if item.Enabled && item.Manifest.Server.StartMode == "automatic" {
			_ = m.Start(context.Background(), item.Manifest.ID)
		}
	}
}

func (m *Manager) Shutdown() {
	m.runtimeMu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.runtimeMu.Unlock()
	for _, id := range ids {
		_ = m.Stop(id)
	}
}
