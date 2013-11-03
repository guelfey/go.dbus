package main

import (
	"fmt"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/exporter"
)

type foo struct {
	__info__ string `interface:"snyh.test.foo",disable:"String,Age"`

	Name string `access:"read"`
	Age  uint32
	hide int

	Singing func(song string, times uint32)
	Cry     func(happy bool, dB float64) `arg1:"happy",arg2:dB"`

	__infoFoo__ byte `args:"name,arg"`
}

func (f *foo) Foo() (string, *dbus.Error) {
	fmt.Println(f)
	return "bar", nil
}

func (f foo) Bar() string {
	return "I will never thraw an dbus.Error"
}

func (f foo) ThrowError1(i uint32) (uint32, *dbus.Error) {
	return 0, &dbus.Error{"BigError", []interface{}{"I'm an big error boom"}}
}

func main() {
	conn, _ := dbus.SessionBus()
	f := foo{
		Name: "snyh",
		Age:  26,
	}
	f.Singing = func(song string, times uint32) {
		fmt.Println("I'm singing the song ", song, times, "times")
	}
	exporter.Export(conn, &f, "com.github.guelfey.Demo", "/com/github/guelfey/Demo", "com.github.guelfey.Demo")
	exporter.Export(conn, f, "com.github.guelfey.Demo", "/com/github/snyh/Demo", "com.github.snyh.Test")
	fmt.Println("Listening on com.github.guelfey.Demo / /com/github/guelfey/Demo ...")
	select {}
}
