package exporter

import "reflect"

import "github.com/guelfey/go.dbus"
import "encoding/xml"
import "bytes"
import "errors"

func getTypeOf(ifc interface{}) (r reflect.Type) {
	r = reflect.TypeOf(ifc)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	return
}

func getValueOf(ifc interface{}) (r reflect.Value) {
	r = reflect.ValueOf(ifc)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	return
}

func GenInterfaceInfo(ifc interface{}) *Interface {
	ifc_info := new(Interface)
	o_type := reflect.TypeOf(ifc)
	n := o_type.NumMethod()

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
		ifc_info.Methods = append(ifc_info.Methods, method)
	}

	// generate properties if any
	if o_type.Kind() == reflect.Ptr {
		o_type = o_type.Elem()
	}
	n = o_type.NumField()
	for i := 0; i < n; i++ {
		field := o_type.Field(i)
		if field.Type.Kind() == reflect.Func {
			ifc_info.Signals = append(ifc_info.Signals, Signal{
				Name: field.Name,
				Args: func() []Arg {
					n := field.Type.NumIn()
					ret := make([]Arg, n)
					for i := 0; i < n; i++ {
						arg := field.Type.In(i)
						ret[i] = Arg{
							Type: dbus.SignatureOfType(arg).String(),
						}
					}
					return ret
				}(),
			})
		} else if field.PkgPath == "" {
			access := field.Tag.Get("access")
			if access != "read" {
				access = "readwrite"
			}
			ifc_info.Properties = append(ifc_info.Properties, Property{
				Name:   field.Name,
				Type:   dbus.SignatureOfType(field.Type).String(),
				Access: access,
			})
		}
	}

	return ifc_info
}

type IntrospectProxy struct {
	infos map[string]interface{}
}

func (i IntrospectProxy) String() string {
	// i.infos reference i so can't use default String()
	ret := "IntrospectProxy ["
	comma := false
	for k, _ := range i.infos {
		if comma {
			ret += ","
		}
		comma = true
		ret += `"` + k + `"`
	}
	ret += "]"
	return ret
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

type PropertiesProxy struct {
	infos map[string]interface{}
}

var errUnknownProperty = dbus.Error{
	"org.freedesktop.DBus.Error.UnknownProperty",
	[]interface{}{"Unknown / invalid Property"},
}
var errUnKnowInterface = dbus.Error{
	"org.freedesktop.DBus.Error.NoSuchInterface",
	[]interface{}{"No such interface"},
}
var errPropertyNotWritable = dbus.Error{
	"org.freedesktop.DBus.Error.NoWritable",
	[]interface{}{"Can't write this property."},
}

func (i PropertiesProxy) GetAll(ifc_name string) map[string]dbus.Variant {
	props := make(map[string]dbus.Variant)
	if ifc, ok := i.infos[ifc_name]; ok {
		o_type := getTypeOf(ifc)
		n := o_type.NumField()
		for i := 0; i < n; i++ {
			field := o_type.Field(i)
			if field.Type.Kind() != reflect.Func && field.PkgPath == "" {
				props[field.Name] = dbus.MakeVariant(getValueOf(ifc).Field(i).Interface())
			}
		}
	}
	return props
}

func (i PropertiesProxy) Set(ifc_name string, prop_name string, value dbus.Variant) *dbus.Error {
	if ifc, ok := i.infos[ifc_name]; ok {
		ifc_t := getTypeOf(ifc)
		t, ok := ifc_t.FieldByName(prop_name)
		v := getValueOf(ifc).FieldByName(prop_name)
		if ok && v.IsValid() {
			if v.CanAddr() && "read" != t.Tag.Get("access") && v.Type() == reflect.TypeOf(value.Value()) {
				v.Set(reflect.ValueOf(value.Value()))
				return nil
			} else {
				return &errPropertyNotWritable
			}
		} else {
			return &errUnknownProperty
		}
	}
	return &errUnKnowInterface
}
func (i PropertiesProxy) Get(ifc_name string, prop_name string) (dbus.Variant, *dbus.Error) {
	if ifc, ok := i.infos[ifc_name]; ok {
		value := getValueOf(ifc).FieldByName(prop_name)
		if value.IsValid() {
			return dbus.MakeVariant(value.Interface()), nil
		} else {
			return dbus.MakeVariant(""), &errUnknownProperty
		}
	} else {
		return dbus.MakeVariant(""), &errUnKnowInterface
	}
}

func Export(c *dbus.Conn, v interface{}, name string, path dbus.ObjectPath, iface string) error {
	not_registered := true
	for _, _name := range c.Names() {
		if _name == name {
			not_registered = false
			break
		}

	}
	if not_registered {
		reply, _ := c.RequestName(name, dbus.NameFlagDoNotQueue)
		if reply != dbus.RequestNameReplyPrimaryOwner {
			return errors.New("name " + name + " already taken")
		}
	}

	err := c.Export(v, path, iface)
	if err != nil {
		return err
	}
	infos := c.GetObjectInfos(path)
	if _, ok := infos["org.freedesktop.DBus.Introspectable"]; !ok {
		infos["org.freedesktop.DBus.Introspectable"] = IntrospectProxy{infos}
	}
	if _, ok := infos["org.freedesktop.DBus.Properties"]; !ok {
		infos["org.freedesktop.DBus.Properties"] = PropertiesProxy{infos}
	}
	return nil
}
