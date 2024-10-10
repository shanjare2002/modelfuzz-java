package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type XraftNodeConfig struct {
	ClusterID 		int
	GroupPort		int
	ServicePort		int
	InterceptorPort int
	SchedulerPort	int
	NodeId			string
	WorkDir			string
	BinaryPath		string
	NumNodes		int
}

type XraftNode struct {
	ID			string
	logger		*Logger
	process		*exec.Cmd
	config		*XraftNodeConfig
	ctx			context.Context
	cancel		func() error

	stdout		*bytes.Buffer
	stderr		*bytes.Buffer
}

func NewXraftNode(config *XraftNodeConfig, logger *Logger) *XraftNode {
	return &XraftNode{
		ID:			config.NodeId,
		logger: 	logger,
		process:	nil,
		config:		config,
		ctx:		nil,
		cancel:		func() error { return nil },
		stdout:		nil,
		stderr: 	nil,
	}
}

func (x *XraftNode) Create() {
	groupConfig := ""
	for i := 1; i <= x.config.NumNodes; i++ {
		if i > 1 {
			groupConfig += " "
		}
		groupConfig += fmt.Sprintf("%d,localhost,%d", i, x.config.GroupPort + i - 1)
	}
	 
	serverArgs := []string {
		"-gc", groupConfig, 
		"-m", "group-member",
		"-i", x.ID,
		"-p2", strconv.Itoa(x.config.ServicePort),
		"-ip", strconv.Itoa(x.config.InterceptorPort),
		"-sp", strconv.Itoa(x.config.SchedulerPort),
		"-d", x.config.WorkDir,
	}
	x.logger.With(LogParams{"server-args": strings.Join(serverArgs, "")}).Debug("Creating server...")

	ctx, cancel := context.WithCancel(context.Background())
	x.process = exec.CommandContext(ctx, x.config.BinaryPath, serverArgs...)

	x.ctx = ctx
	x.cancel = func ()  error {
		err := x.process.Process.Signal(os.Interrupt)
		cancel()
		return err
	}

	if x.stdout == nil {
		x.stdout = new(bytes.Buffer)
	}
	if x.stderr == nil {
		x.stderr = new(bytes.Buffer)
	}
	x.process.Stdout = x.stdout
	x.process.Stderr = x.stderr
	x.process.Cancel = x.cancel
}

func (x *XraftNode) Start() error {
	if x.ctx == nil || x.process == nil {
		return errors.New("Xraft server not started.")
	}

	x.Create()
	return x.process.Start()
}

func (x *XraftNode) Cleanup() {
	os.RemoveAll(x.config.WorkDir)
}

func (x *XraftNode) Stop() error {
	if x.ctx == nil || x.process == nil {
		return errors.New("Xraft server not started.")
	}
	select {
	case <- x.ctx.Done():
		return errors.New("Xraft server already stopped.")
	default:
	}
	x.cancel()

	done := make(chan error, 1)
	go func() {
		err := x.process.Wait()
		done <- err
	}()

	var err error = nil
	select {
	case <- time.After(50 * time.Millisecond):
		err = x.process.Process.Kill()
	case err = <- done:
	}

	x.ctx = nil
	x.cancel = func() error { return nil }
	x.process = nil

	return err
}

func (x *XraftNode) GetLogs() (string, string) {
	if x.stdout == nil || x.stderr == nil {
		return "", ""
	}

	return x.stdout.String(), x.stderr.String()
}

// func (x *XraftNode) Execute() error {
// 	if x.ctx == nil || x.process == nil {
// 		return errors.New("Xraft server not started.")
// 	}
// 	select {
// 	case <- x.ctx.Done():
// 		return errors.New("Xraft server already stopped.")
// 	default:
// 	}
	
// 	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
// 	defer cancel()
// 	// TODO - client call
// 	return nil
// }

// func (x *XraftNode) ExecuteAsync() error {
// 	if x.ctx == nil || x.process == nil {
// 		return errors.New("Xraft server not started.")
// 	}
// 	select {
// 	case <- x.ctx.Done():
// 		return errors.New("Xraft server already stopped.")
// 	default:
// 	}
	
// 	go func ()  {
// 		ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
// 		defer cancel()
// 		// TODO - client call	
// 	}()
// 	return nil
// }

type ClusterConfig struct {
	FuzzerType			string // TODO
	ClusterID			int
	NumNodes			int
	XraftBinaryPath		string
	XraftClientPath		string
	BaseGroupPort		int
	BaseServicePort		int
	BaseInterceptorPort int
	BaseSchedulerPort	int
	WorkDir				string
	LogLevel			string
}

func (c *ClusterConfig) Copy() *ClusterConfig {
	return &ClusterConfig{
		NumNodes: c.NumNodes,
		XraftBinaryPath: c.XraftBinaryPath,
		XraftClientPath: c.XraftClientPath,
		BaseGroupPort: c.BaseGroupPort,
		BaseServicePort: c.BaseServicePort,
		BaseInterceptorPort: c.BaseInterceptorPort,
		BaseSchedulerPort: c.BaseSchedulerPort,
		WorkDir: c.WorkDir,
		LogLevel: c.LogLevel,
	}
}

func (c *ClusterConfig) SetDefaults() {
	if c.XraftBinaryPath == "" {
		c.XraftBinaryPath = "../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore"
	}
	if c.XraftClientPath == "" {
		c.XraftClientPath = "../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore-cli"
	}
	if c.WorkDir == "" {
		c.WorkDir = fmt.Sprintf("output/%s/tmp/%d", c.FuzzerType, c.ClusterID)
	}
	os.MkdirAll(c.WorkDir, 0777)

	if c.LogLevel == "" {
		c.LogLevel = "INFO"
	}
}

func (c*ClusterConfig) GetNodeConfig(id string) *XraftNodeConfig {
	nodeWorkDir := path.Join(c.WorkDir, id)
	if _, err := os.Stat(nodeWorkDir); err == nil {
		os.RemoveAll(nodeWorkDir)
	}
	os.MkdirAll(nodeWorkDir, 0777)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil
	}

	return &XraftNodeConfig{
		ClusterID:			c.ClusterID, 		
		GroupPort:			c.BaseGroupPort + idInt,	
		ServicePort:		c.BaseServicePort + idInt,		
		InterceptorPort:	c.BaseInterceptorPort + idInt,
		SchedulerPort:		c.BaseSchedulerPort,
		NodeId:				id,
		WorkDir: 			nodeWorkDir,
		BinaryPath:			c.XraftBinaryPath,
		NumNodes: 			c.NumNodes,	
	}
}

type Cluster struct {
	Nodes	map[string]*XraftNode
	Config	*ClusterConfig
	Logger	*Logger
	Client  *XraftClient
}

func NewCluster(config *ClusterConfig, logger *Logger) *Cluster {
	config.SetDefaults()
	c := &Cluster{
		Config: config,
		Nodes: make(map[string]*XraftNode),
		Logger: logger,
		Client: NewXraftClient(config.NumNodes, config.BaseServicePort, config.XraftClientPath),
	}

	for i := 1; i <= config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(strconv.Itoa(i))
		c.Nodes[strconv.Itoa(i)] = NewXraftNode(nConfig, c.Logger.With(LogParams{"node": i}))
	}
	return c
}

func (c *Cluster) Start() error {
	for i := 1; i <= c.Config.NumNodes; i++ {
		node := c.Nodes[strconv.Itoa(i)]
		if err := node.Start(); err != nil {
			return fmt.Errorf("Error starting node %d: %s", i, err)
		}
	}
	return nil
}

func (c *Cluster) Destroy() error {
	var err error = nil
	for _, node := range c.Nodes {
		err = node.Stop()
		node.Cleanup()
	}
	return err
}

func (c *Cluster) GetNode(nodeId string) (*XraftNode, bool) {
	node, ok := c.Nodes[nodeId]
	return node, ok

}

func (c *Cluster) GetLogs() string {
	logLines := []string{}
	for nodeID, node := range c.Nodes {
		logLines = append(logLines, fmt.Sprintf("Logs for node: %s\n", nodeID))
		stdout, stderr := node.GetLogs()
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr, "\n\n")
	}
	return strings.Join(logLines, "\n")
}

func (c *Cluster) SendRequest() {
	clientArgs := []string {
		"-gc", c.Client.GroupConfig,
		"-ic", "\"kvstore-set x 1\"",
	}
	process := exec.Command(c.Client.ClientBinary, clientArgs...)

	cmdDone := make(chan error, 1)
	go func ()  {
		err := process.Start()
		cmdDone <- err
	}()

	select {
	case <- time.After(2 * time.Second):
		syscall.Kill(-process.Process.Pid, syscall.SIGKILL)
	case <- cmdDone:
	}
}

type XraftClient struct {
	ClientBinary	string
	GroupConfig		string
}

func NewXraftClient(numNodes int, baseServicePort int, clientBinary string) *XraftClient{
	groupConfig := ""
	for i := 1; i <= numNodes; i++ {
		if i > 1 {
			groupConfig += " "
		}
		groupConfig += fmt.Sprintf("%d,localhost,%d", i, baseServicePort + i -1)
	}

	return &XraftClient{
		GroupConfig: groupConfig,
		ClientBinary: clientBinary,
	}
}