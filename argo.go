// Package argo is an RDF manipulation, parsing and serialisation library.
package argo

import (
	"io"
	"strings"
)

// A Parser is a function that can parse a particular representation of RDF and stream the triples
// on a channel.
type Parser func(io.Reader, chan *Triple, chan error)

// A Serializer is a function that recieves triples sent along a channel and serializes them into a
// particular representation of RDF.
type Serializer func(io.Writer, chan *Triple, chan error)

// A Store is a container for RDF triples. For example, it could be backed by a flat list or a
// relational database.
type Store interface {
	// Function Add adds the given triple to the store and returns its index.
	Add(*Triple) int

	// Function Remove removes the given triple from the store.
	Remove(*Triple)

	// Function Remove removes triple with the given index from the store.
	RemoveIndex(int)

	// Function Clear removes all triples from the store.
	Clear()

	// Function Num returns the number of triples in the store.
	Num() int

	// Function IterTriples returns a channel that will yield the triples of the store. The channel
	// will be closed when iteration is completed.
	IterTriples() chan *Triple
}

// Function splitPrefix takes a given URI and splits it into a base URI and a local name (suitable
// for using as a qname in XML).
func splitPrefix(uri string) (base string, name string) {
	index := strings.LastIndex(uri, "#") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	index = strings.LastIndex(uri, "/") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	return "", uri
}