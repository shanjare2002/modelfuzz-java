package main

import (
	"bytes"
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
	BaseGroupPort	int
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

	stdout		*bytes.Buffer
	stderr		*bytes.Buffer
}

func NewXraftNode(config *XraftNodeConfig, logger *Logger) *XraftNode {
	return &XraftNode{
		ID:			config.NodeId,
		logger: 	logger,
		process:	nil,
		config:		config,
		stdout:		nil,
		stderr: 	nil,
	}
}

func (x *XraftNode) Create() {
	serverArgs := []string { 
		x.config.BinaryPath,
		"-m", "group-member",
		"-i", x.ID,
		"-p2", strconv.Itoa(x.config.ServicePort),
		"-ip", strconv.Itoa(x.config.InterceptorPort),
		"-sp", strconv.Itoa(x.config.SchedulerPort),
		"-d", x.config.WorkDir,
		"-gc",
	}
	for i := 1; i <= x.config.NumNodes; i++ {
		serverArgs = append(serverArgs, fmt.Sprintf("%d,localhost,%d", i, x.config.BaseGroupPort + i))
	}
	x.logger.With(LogParams{"server-args": strings.Join(serverArgs, "")}).Debug("Creating server...")

	x.process = exec.Command("bash", serverArgs...)
	x.process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
		return errors.New("Xraft server not started.")
	}
	return x.process.Start()
}

func (x *XraftNode) Cleanup() {
	os.RemoveAll(x.config.WorkDir)
}

func (x *XraftNode) Stop() error {
	x.logger.Debug("Stopping node...")
	if x.process == nil {
		return errors.New("Xraft server not started.")
	}
	// done := make(chan error, 1)
	// go func() {
	// 	err := x.process.Wait()
	// 	done <- err
	// }()

	// var err error = nil
	// select {
	// case <- time.After(50 * time.Millisecond):
	// 	err = x.process.Process.Kill()
	// case err = <- done:
	// }
	
	var err error
	if x.process.Process != nil {
		err = syscall.Kill(-x.process.Process.Pid, syscall.SIGKILL)
	}

	x.process = nil

	return err
}

func (x *XraftNode) GetLogs() (string, string) {
	if x.stdout == nil || x.stderr == nil {
		x.logger.Debug("Nil stdout or stderr.")
		return "", ""
	}

	return x.stdout.String(), x.stderr.String()
}

type ClusterConfig struct {
	FuzzerType			FuzzerType
	ClusterID			int
	NumNodes			int
	XraftBinaryPath		string
	XraftClientPath		string
	BaseGroupPort		int
	BaseServicePort		int
	BaseInterceptorPort int
	SchedulerPort		int
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
		SchedulerPort: c.SchedulerPort,
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
		c.WorkDir = fmt.Sprintf("output/%s/tmp/%d", c.FuzzerType.String(), c.ClusterID)
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
		BaseGroupPort:  	c.BaseGroupPort,
		ServicePort:		c.BaseServicePort + idInt,		
		InterceptorPort:	c.BaseInterceptorPort + idInt,
		SchedulerPort:		c.SchedulerPort,
		NodeId:				id,
		WorkDir: 			nodeWorkDir,
		BinaryPath:			c.XraftBinaryPath,
		NumNodes: 			c.NumNodes,	
	}
}

type Cluster struct {
	Nodes	map[string]*XraftNode
	Config	*ClusterConfig
	logger	*Logger
	Client  *XraftClient
}

func NewCluster(config *ClusterConfig, logger *Logger) *Cluster {
	config.SetDefaults()
	c := &Cluster{
		Config: config,
		Nodes: make(map[string]*XraftNode),
		logger: logger,
		Client: NewXraftClient(config.NumNodes, config.BaseServicePort, config.XraftClientPath),
	}

	for i := 1; i <= config.NumNodes; i++ {
		nConfig := config.GetNodeConfig(strconv.Itoa(i))
		c.Nodes[strconv.Itoa(i)] = NewXraftNode(nConfig, c.logger.With(LogParams{"node": i}))
	}
	c.logger.Debug("Initialized cluster.")
	return c
}

func (c *Cluster) Start() error {
	c.logger.Debug("Starting cluster...")
	for i := 1; i <= c.Config.NumNodes; i++ {
		node := c.Nodes[strconv.Itoa(i)]
		node.Start()
		// if err := node.Start(); err != nil {
		// 	return fmt.Errorf("Error starting node %d: %s", i, err)
		// }
	}
	return nil
}

func (c *Cluster) Destroy() error {
	c.logger.Debug("Destroying cluster...")
	var err error = nil
	for _, node := range c.Nodes {
		err = node.Stop()
		node.Cleanup()
	}

	os.RemoveAll(c.Config.WorkDir)
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
		logLines = append(logLines, "----- Stdout -----", stdout, "----- Stderr -----", stderr)
	}
	return strings.Join(logLines, "\n")
}

func (c *Cluster) SendRequest() {
	c.logger.Debug("Sending client request...")
	clientArgs := []string {
		c.Config.XraftClientPath,
		"-gc",
	}
	for i := 1; i <= c.Config.NumNodes; i++ {
		clientArgs = append(clientArgs, fmt.Sprintf("%d,localhost,%d", i, c.Client.BaseServicePort + i))
	}

	process := exec.Command("bash", clientArgs...)

	// cmdDone := make(chan error, 1)
	process.Start()

	select {
	case <- time.After(2 * time.Second):
		syscall.Kill(-process.Process.Pid, syscall.SIGKILL)
	default:
	}
}

type XraftClient struct {
	ClientBinary	string
	BaseServicePort	int
}

func NewXraftClient(numNodes int, baseServicePort int, clientBinary string) *XraftClient{
	return &XraftClient{
		BaseServicePort: baseServicePort,
		ClientBinary: clientBinary,
	}
}