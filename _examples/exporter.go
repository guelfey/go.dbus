package main

import (
	"fmt"
	"github.com/guelfey/go.dbus"
	"github.com/guelfey/go.dbus/exporter"
	"os"
	"runtime/pprof"
	"time"
)

type foo struct {
	__info__ string `interface:"snyh.test.foo",disable:"String,Age"`

	Name string `access:"read"`
	Age  uint32
	hide int

	Singing func(song string)
	Cry     func(happy bool, dB float64) `arg1:"happy",arg2:dB"`

	__infoFoo__ byte `args:"name,arg"`
}

func (f foo) Foo() (string, *dbus.Error) {
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
	file, _ := os.Create("p.txt")
	pprof.StartCPUProfile(file)
	defer pprof.StopCPUProfile()
	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	reply, err := conn.RequestName("com.github.guelfey.Demo",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}
	f := foo{
		Name: "snyh",
		Age:  26,
	}
	exporter.Export(conn, &f, "/com/github/guelfey/Demo", "com.github.guelfey.Demo")
	fmt.Println("Listening on com.github.guelfey.Demo / /com/github/guelfey/Demo ...")
	select {
	case <-time.After(time.Second * 5):
		/*return*/
	}
	select {}
}
