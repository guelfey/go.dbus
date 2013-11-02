// Package introspect provides some utilities for dealing with the DBus
// introspection format.
package exporter

import "encoding/xml"

// Node is the root element of an introspection.
type Node struct {
	XMLName    xml.Name    `xml:"node"`
	Name       string      `xml:"name,attr,omitempty"`
	Interfaces []Interface `xml:"interface"`
	Children   []Node      `xml:"node,omitempty"`
}

// Interface describes a DBus interface that is available on the message bus.
type Interface struct {
	Name        string       `xml:"name,attr"`
	Methods     []Method     `xml:"method"`
	Signals     []Signal     `xml:"signal"`
	Properties  []Property   `xml:"property"`
	Annotations []Annotation `xml:"annotation"`
}

// Method describes a Method on an Interface as retured by an introspection.
type Method struct {
	Name        string       `xml:"name,attr"`
	Args        []Arg        `xml:"arg"`
	Annotations []Annotation `xml:"annotation"`
}

// Signal describes a Signal emitted on an Interface.
type Signal struct {
	Name        string       `xml:"name,attr"`
	Args        []Arg        `xml:"arg"`
	Annotations []Annotation `xml:"annotation"`
}

// Property describes a property of an Interface.
type Property struct {
	Name        string       `xml:"name,attr"`
	Type        string       `xml:"type,attr"`
	Access      string       `xml:"access,attr"`
	Annotations []Annotation `xml:"annotation"`
}

// Arg represents an argument of a method or a signal.
type Arg struct {
	Name      string `xml:"name,attr,omitempty"`
	Type      string `xml:"type,attr"`
	Direction string `xml:"direction,attr,omitempty"`
}

// Annotation is an annotation in the introspection format.
type Annotation struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}
