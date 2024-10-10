package main

type EventNode struct {
	Event
	Node string
	Prev string
	ID   string `json:"-"`
}

type Guider interface {
	Check(iter string, trace *Trace, eventTrace *EventTrace, record bool) (bool, int)
	Coverage() int
	Reset()
}