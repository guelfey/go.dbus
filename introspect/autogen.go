package introspect

import "reflect"
import "github.com/guelfey/go.dbus"
import "encoding/xml"
import "bytes"

func GenInterfaceInfo(ifc interface{}) *Interface {
	ifc_info := new(Interface)
	o_type := reflect.TypeOf(ifc)
	n := o_type.NumMethod()
	r := make([]Method, 0)

	for i := 0; i < n; i++ {
		name := o_type.Method(i).Name
		method := Method{}
		method.Name = name

		m := o_type.Method(i).Type
		n_in := m.NumIn()
		n_out := m.NumOut()
		args := make([]Arg, 0)
		//Method's first paramter is the struct which this method bound to.
		for i := 1; i < n_in; i++ {
			args = append(args, Arg{
				Type:      dbus.SignatureOfType(m.In(i)).String(),
				Direction: "in",
			})
		}
		for i := 0; i < n_out; i++ {
			if m.Out(i) != reflect.TypeOf(&dbus.Error{}) {
				args = append(args, Arg{
					Type:      dbus.SignatureOfType(m.Out(i)).String(),
					Direction: "out",
				})
			}
		}
		method.Args = args
		r = append(r, method)
	}
	ifc_info.Methods = r
	return ifc_info
}

type IntrospectProxy struct {
	infos map[string]interface{}
}

func (i IntrospectProxy) Introspect() (string, *dbus.Error) {
	var node = new(Node)
	for name, ifc := range i.infos {
		info := GenInterfaceInfo(ifc)
		info.Name = name
		node.Interfaces = append(node.Interfaces, *info)
	}
	var buffer bytes.Buffer

	writer := xml.NewEncoder(&buffer)
	writer.Indent("", "     ")
	writer.Encode(node)
	return buffer.String(), nil
}

func Export(c *dbus.Conn, v interface{}, path dbus.ObjectPath, iface string) error {
	err := c.Export(v, path, iface)
	if err != nil {
		return err
	}
	infos := c.GetObjectInfos(path)
	if _, ok := infos["org.freedesktop.DBus.Introspectable"]; !ok {
		infos["org.freedesktop.DBus.Introspectable"] = IntrospectProxy{infos}
	}
	//TODO export Properties
	return nil
}
