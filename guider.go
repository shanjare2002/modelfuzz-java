package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Guider interface {
	Check(iter string, trace *Trace, eventTrace *EventTrace, record bool) (bool, int, int, int)
	Coverage() int
	TransitionCoverage() int
	Reset()
}

func NewGuider(fuzzerType FuzzerType, addr, recordPath string, jacocoFile string, jacocoOutput string) Guider {
	if fuzzerType == ModelFuzz || fuzzerType == RandomFuzzer {
		return NewTLCStateGuider(addr, recordPath, jacocoFile, jacocoOutput)
	} else if fuzzerType == TraceFuzzer {
		return NewTraceCoverageGuider(addr, recordPath, jacocoFile, jacocoOutput)
	} else {
		return nil
	}
}

type TLCStateGuider struct {
	TLCAddr          string
	statesMap        map[int64]bool
	tlcClient        *TLCClient
	stateTransitions map[int64][]int64
	// objectPath      string
	// gCovProgramPath string

	recordPath   string
	jacocoFile   string
	jacocoOutput string
}

var _ Guider = &TLCStateGuider{}

func NewTLCStateGuider(tlcAddr, recordPath string, jacocoFile string, jacocoOutput string) *TLCStateGuider {
	return &TLCStateGuider{
		TLCAddr:          tlcAddr,
		statesMap:        make(map[int64]bool),
		tlcClient:        NewTLCClient(tlcAddr),
		stateTransitions: make(map[int64][]int64),
		recordPath:       recordPath,
		jacocoFile:       jacocoFile,
		jacocoOutput:     jacocoOutput,
	}
}

func (t *TLCStateGuider) Reset() {
	t.statesMap = make(map[int64]bool)
	// clearCovData(t.objectPath)
}

func (t *TLCStateGuider) Coverage() int {
	return len(t.statesMap)
}

func (t *TLCStateGuider) TransitionCoverage() int {
	return len(t.stateTransitions)
}

func (t *TLCStateGuider) Check(iter string, trace *Trace, eventTrace *EventTrace, record bool) (bool, int, int, int) {

	numNewStates := 0
	numNewTransitions := 0
	numNewLines := 0
	if tlcStates, err := t.tlcClient.SendTrace(eventTrace); err == nil {
		if record {
			t.recordTrace(iter, trace, eventTrace, tlcStates)
		}

		// Update states and transitions
		for _, s := range tlcStates {
			_, ok := t.statesMap[s.Key]
			if !ok {
				numNewStates += 1
				t.statesMap[s.Key] = true
			}
		}

		start := true
		previous_state := int64(-1)
		for _, s := range tlcStates {
			if start {
				previous_state = s.Key
				start = false
			} else {
				prevKey := previous_state
				currKey := s.Key
				exists := false
				for _, v := range t.stateTransitions[prevKey] {
					if v == currKey {
						exists = true
						break
					}
				}
				if !exists {
					t.stateTransitions[prevKey] = append(t.stateTransitions[prevKey], currKey)
					numNewTransitions += 1
				}
				previous_state = currKey
			}
		}

		if t.jacocoOutput != "" {
			// Generate XML report if jacocoFile and jacocoOutput are provided
			if err := t.generateXMLReport(); err != nil {
				fmt.Printf("failed to generate XML report: %v", err)
			}
			numNewLines, err = parseCoverageAndUpdate(t.jacocoOutput)
			if err != nil {
				fmt.Printf("failed to parse coverage: %v", err)
			}
		}
	}

	return numNewStates != 0, numNewStates, numNewTransitions, numNewLines
}

func (t *TLCStateGuider) recordTrace(as string, trace *Trace, eventTrace *EventTrace, states []TLCState) {

	filePath := path.Join(t.recordPath, as+".json")
	data := map[string]interface{}{
		"trace":       trace,
		"event_trace": eventTrace,
		"state_trace": parseTLCStateTrace(states),
	}
	dataB, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.Write(dataB)
	writer.Flush()
}

func parseTLCStateTrace(states []TLCState) []TLCState {
	newStates := make([]TLCState, len(states))
	for i, s := range states {
		repr := strings.ReplaceAll(s.Repr, "\n", ",")
		repr = strings.ReplaceAll(repr, "/\\", "")
		repr = strings.ReplaceAll(repr, "\u003e\u003e", "]")
		repr = strings.ReplaceAll(repr, "\u003c\u003c", "[")
		repr = strings.ReplaceAll(repr, "\u003e", ">")
		newStates[i] = TLCState{
			Repr: repr,
			Key:  s.Key,
		}
	}
	return newStates
}

type TraceCoverageGuider struct {
	traces map[string]bool
	*TLCStateGuider
}

var _ Guider = &TraceCoverageGuider{}

func NewTraceCoverageGuider(tlcAddr, recordPath string, jacocoFile string, jacocoOutput string) *TraceCoverageGuider {
	return &TraceCoverageGuider{
		traces:         make(map[string]bool),
		TLCStateGuider: NewTLCStateGuider(tlcAddr, recordPath, jacocoFile, jacocoOutput),
	}
}

func (t *TraceCoverageGuider) Check(iter string, trace *Trace, events *EventTrace, record bool) (bool, int, int, int) {
	t.TLCStateGuider.Check(iter, trace, events, record)

	eTrace := newEventTrace(events)
	key := eTrace.Hash()

	new := 0
	if _, ok := t.traces[key]; !ok {
		t.traces[key] = true
		new = 1
	}
	return new != 0, new, 0, 0
}

func (t *TraceCoverageGuider) Coverage() int {
	return t.TLCStateGuider.Coverage()
}

func (t *TraceCoverageGuider) Reset() {
	t.traces = make(map[string]bool)
	t.TLCStateGuider.Reset()
}

type eventTrace struct {
	Nodes map[string]*eventNode
}

func (e *eventTrace) Hash() string {
	bs, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

type eventNode struct {
	Event
	Node string
	Prev string
	ID   string `json:"-"`
}

func (e *eventNode) Hash() string {
	bs, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func newEventTrace(events *EventTrace) *eventTrace {
	eTrace := &eventTrace{
		Nodes: make(map[string]*eventNode),
	}
	curEvent := make(map[string]*eventNode)

	for _, e := range events.Events {
		node := &eventNode{
			Event: e.Copy(),
			Node:  e.Node,
			Prev:  "",
		}
		prev, ok := curEvent[e.Node]
		if ok {
			node.Prev = prev.ID
		}
		node.ID = node.Hash()
		curEvent[e.Node] = node
		eTrace.Nodes[node.ID] = node
	}
	return eTrace
}

func (t *TLCStateGuider) generateXMLReport() error {
	cmd := exec.Command("java", "-jar", "jacococli.jar", "report", t.jacocoFile,
		"--classfiles", "../xraft-controlled/xraft-core/target/classes",
		"--classfiles", "../xraft-controlled/xraft-kvstore/target/classes",
		"--sourcefiles", "../xraft-controlled/xraft-core/src/main/java",
		"--sourcefiles", "../xraft-controlled/xraft-kvstore/src/main/java",
		"--xml", t.jacocoOutput)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type Line struct {
	Number          int `xml:"nr,attr"`
	MissedInstr     int `xml:"mi,attr"`
	CoveredInstr    int `xml:"ci,attr"`
	MissedBranches  int `xml:"mb,attr"`
	CoveredBranches int `xml:"cb,attr"`
}

// ------- Function for code coverage -------
var coverageData = map[string]map[int]struct{}{}

type SourceFile struct {
	Name  string `xml:"name,attr"`
	Lines []Line `xml:"line"`
}

type Package struct {
	Name        string       `xml:"name,attr"`
	SourceFiles []SourceFile `xml:"sourcefile"`
}

type Report struct {
	Packages []Package `xml:"package"`
}

func parseCoverageAndUpdate(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var report Report
	if err := xml.NewDecoder(f).Decode(&report); err != nil {
		return 0, err
	}

	newLines := 0

	for _, pkg := range report.Packages {
		for _, src := range pkg.SourceFiles {
			filePath := filepath.Join(pkg.Name, src.Name)

			// Ensure map for this file exists
			if _, ok := coverageData[filePath]; !ok {
				coverageData[filePath] = map[int]struct{}{}
			}

			for _, line := range src.Lines {
				if line.CoveredInstr > 0 {
					if _, already := coverageData[filePath][line.Number]; !already {
						newLines += 1
						coverageData[filePath][line.Number] = struct{}{}
					}
				}
			}
		}
	}
	return newLines, nil
}

func CoverageDataLength() int {
	count := 0
	for _, lines := range coverageData {
		count += len(lines)
	}
	return count
}
