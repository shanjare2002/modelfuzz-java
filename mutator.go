package main

type Mutator interface {
	Mutate(*Trace, *EventTrace) (*Trace, bool)
}