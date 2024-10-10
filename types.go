package main

type Choice struct {
	Type	string
	Node	string
	From 	string
	To		string
}

type Trace struct {
	Choices []Choice
}

type Event struct {
	Name   string
	Node   int `json:"-"`
	Params map[string]interface{}
	Reset  bool
}

type EventTrace struct {
	Nodes	map[string]*EventNode
}

type Stats struct {
	Coverages           []int
	RandomTraces        int
	MutatedTraces       int
}