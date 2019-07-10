package main

import (
	"fmt"
	"github.com/keybase/go.dbus"
	"github.com/keybase/go.dbus/introspect"
	"os"
)

const intro = `
<node>
	<interface name="com.github.keybase.Demo">
		<method name="Foo">
			<arg direction="out" type="s"/>
		</method>
	</interface>` + introspect.IntrospectDataString + `</node> `

type foo string

func (f foo) Foo() (string, *dbus.Error) {
	fmt.Println(f)
	return string(f), nil
}

func main() {
	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	reply, err := conn.RequestName("com.github.keybase.Demo",
		dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}
	f := foo("Bar!")
	conn.Export(f, "/com/github/keybase/Demo", "com.github.keybase.Demo")
	conn.Export(introspect.Introspectable(intro), "/com/github/keybase/Demo",
		"org.freedesktop.DBus.Introspectable")
	fmt.Println("Listening on com.github.keybase.Demo / /com/github/keybase/Demo ...")
	select {}
}
