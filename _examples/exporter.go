package main

import (
	"fmt"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/exporter"
	"os"
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
	reply, _ := conn.RequestName("com.github.guelfey.Demo",
		dbus.NameFlagDoNotQueue)
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}
	f := foo{
		Name: "snyh",
		Age:  26,
	}
	f.Singing = func(song string, times uint32) {
		fmt.Println("I'm singing the song ", song, times, "times")
	}
	exporter.Export(conn, &f, "/com/github/guelfey/Demo", "com.github.guelfey.Demo")
	conn.Emit("/com/githu/guelfey/Demo", "com.github.guelfey.Demo.Singing", "simle forver")
	fmt.Println("Listening on com.github.guelfey.Demo / /com/github/guelfey/Demo ...")
	select {}
}
