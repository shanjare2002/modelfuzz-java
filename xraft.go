package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type XraftNode struct {
	ID      string
	logger  *Logger
	process *exec.Cmd
	config  *NodeConfig

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func NewXraftNode(config *NodeConfig, logger *Logger) *XraftNode {
	return &XraftNode{
		ID:      config.NodeId,
		logger:  logger,
		process: nil,
		config:  config,
		stdout:  nil,
		stderr:  nil,
	}
}

func (x *XraftNode) Create() {
	serverArgs := []string{
		x.config.ServerPath,
		"-m", "group-member",
		"-i", x.ID,
		"-p2", strconv.Itoa(x.config.ServicePort),
		"-ip", strconv.Itoa(x.config.InterceptorPort),
		"-sp", strconv.Itoa(x.config.SchedulerPort),
		"-d", x.config.WorkDir,
		"-gc",
	}
	for i := 1; i <= x.config.NumNodes; i++ {
		serverArgs = append(serverArgs, fmt.Sprintf("%d,localhost,%d", i, x.config.BaseGroupPort+i))
	}
	x.logger.With(LogParams{"server-args": strings.Join(serverArgs, "")}).Debug("Creating server...")

	x.process = exec.Command("bash", serverArgs...)
	x.process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	env := os.Environ()

	// Make Jacoco file unique per node id
	cwd, err := os.Getwd()
	if err != nil {
		x.logger.Debug("Failed to get current working directory")
	} else {
		x.logger.Debug("Current working directory: " + cwd)
	}
	jacocoFile := fmt.Sprintf("%s/output/modelfuzz/jacoco/jacocoRun.exec", cwd)
	jacocoEnv := fmt.Sprintf("-javaagent:%s/jacocoagent.jar=output=file,destfile=%s,append=true,dumponexit=true", cwd, jacocoFile)
	env = append(env, "JAVA_TOOL_OPTIONS="+jacocoEnv)
	x.process.Env = env

	if x.stdout == nil {
		x.stdout = new(bytes.Buffer)
	}
	if x.stderr == nil {
		x.stderr = new(bytes.Buffer)
	}
	x.process.Stdout = x.stdout
	x.process.Stderr = x.stderr
}

func (x *XraftNode) Start() error {
	x.logger.Debug("Starting node...")
	x.Create()
	if x.process == nil {
		return errors.New("xraft server not started")
	}
	return x.process.Start()
}

func (x *XraftNode) Cleanup() {
	os.RemoveAll(x.config.WorkDir)
}

func (x *XraftNode) Stop() error {
	x.logger.Debug("Stopping node...")
	if x.process == nil || x.process.Process == nil {
		return errors.New("xraft server not started")
	}

	err := syscall.Kill(-x.process.Process.Pid, syscall.SIGTERM)
	if err != nil {
		x.logger.Debug("SIGTERM failed, trying SIGKILL")
		_ = syscall.Kill(-x.process.Process.Pid, syscall.SIGKILL)
		x.process = nil
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- x.process.Wait()
	}()

	select {
	case err := <-done:
		x.process = nil
		return err
	case <-time.After(20 * time.Second):
		x.logger.Debug("Process still running, sending SIGKILL")
		_ = syscall.Kill(-x.process.Process.Pid, syscall.SIGKILL)
		x.process = nil
		return errors.New("process did not terminate in time")
	}
}

func (x *XraftNode) GetLogs() (string, string) {
	if x.stdout == nil || x.stderr == nil {
		x.logger.Debug("Nil stdout or stderr.")
		return "", ""
	}

	return x.stdout.String(), x.stderr.String()
}

type XraftClient struct {
	ClientBinary    string
	BaseServicePort int
	logger          *Logger
	NumNodes        int
}

func NewXraftClient(numNodes int, baseServicePort int, clientBinary string, logger *Logger) *XraftClient {
	return &XraftClient{
		BaseServicePort: baseServicePort,
		ClientBinary:    clientBinary,
		logger:          logger,
		NumNodes:        numNodes,
	}
}

func (c *XraftClient) SendRequest() {
	c.logger.Debug("Sending client request...")
	clientArgs := []string{
		c.ClientBinary,
		"-gc",
	}
	for i := 1; i <= c.NumNodes; i++ {
		clientArgs = append(clientArgs, fmt.Sprintf("%d,localhost,%d", i, c.BaseServicePort+i))
	}

	process := exec.Command("bash", clientArgs...)
	env := os.Environ()

	cwd, err := os.Getwd()
	if err != nil {
		c.logger.Debug("Failed to get current working directory")
	} else {
		c.logger.Debug("Current working directory: " + cwd)
	}
	jacocoFile := fmt.Sprintf("%s/output/modelfuzz/jacoco/jacocoRun.exec", cwd)
	jacocoEnv := fmt.Sprintf("-javaagent:%s/jacocoagent.jar=output=file,destfile=%s,append=true,dumponexit=true", cwd, jacocoFile)
	env = append(env, "JAVA_TOOL_OPTIONS="+jacocoEnv)
	process.Env = env

	process.Start()

	select {
	case <-time.After(2 * time.Second):
		c.logger.Debug("Client request timed out, sending SIGKILL")
		syscall.Kill(-process.Process.Pid, syscall.SIGKILL)
	default:
	}
}
