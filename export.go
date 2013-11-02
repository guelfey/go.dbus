package dbus

import (
	"errors"
	"reflect"
	"strings"
	"unicode"
)

var (
	errmsgInvalidArg = Error{
		"org.freedesktop.DBus.Error.InvalidArgs",
		[]interface{}{"Invalid type / number of args"},
	}
	errmsgNoObject = Error{
		"org.freedesktop.DBus.Error.NoSuchObject",
		[]interface{}{"No such object"},
	}
	errmsgUnknownMethod = Error{
		"org.freedesktop.DBus.Error.UnknownMethod",
		[]interface{}{"Unknown / invalid method"},
	}
)

func exportedMethod(v interface{}, name string) reflect.Value {
	if v == nil {
		return reflect.Value{}
	}
	m := reflect.ValueOf(v).MethodByName(name)
	if !m.IsValid() {
		return reflect.Value{}
	}
	return m
}

func handleIntrospectionPartialPathRequest(possible_path []string, partial_path string) string {
	var xml string = `<node>`
	valid_field := make(map[string]bool)
	for _, path := range possible_path {
		begin := strings.Index(path, partial_path)
		if begin != -1 {
			path = path[begin+len(partial_path):]
			if path[0] == '/' {
				path = path[1:]
			}
			end := strings.Index(path, "/")
			if end != -1 {
				path = path[:end]
			}
			valid_field[path] = true
		}
	}
	for k, _ := range valid_field {
		xml += `	<node name="` + k + `"/>`
	}
	xml += `</node>`
	return xml
}

// handleCall handles the given method call (i.e. looks if it's one of the
// pre-implemented ones and searches for a corresponding handler if not).
func (conn *Conn) handleCall(msg *Message) {
	name := msg.Headers[FieldMember].value.(string)
	path := msg.Headers[FieldPath].value.(ObjectPath)
	ifaceName, hasIface := msg.Headers[FieldInterface].value.(string)
	sender := msg.Headers[FieldSender].value.(string)
	serial := msg.serial
	if ifaceName == "org.freedesktop.DBus.Peer" {
		switch name {
		case "Ping":
			conn.sendReply(sender, serial)
		case "GetMachineId":
			conn.sendReply(sender, serial, conn.uuid)
		default:
			conn.sendError(errmsgUnknownMethod, sender, serial)
		}
		return
	} else if _, ok := conn.handlers[path]; !ok && ifaceName == "org.freedesktop.DBus.Introspectable" && name == "Introspect" {
		paths := make([]string, 0)
		for key, _ := range conn.handlers {
			paths = append(paths, string(key))
		}
		conn.sendReply(sender, serial, handleIntrospectionPartialPathRequest(paths, string(path)))
		return
	}
	if len(name) == 0 || unicode.IsLower([]rune(name)[0]) {
		conn.sendError(errmsgUnknownMethod, sender, serial)
		return
	}

	var m reflect.Value
	if hasIface {
		conn.handlersLck.RLock()
		obj, ok := conn.handlers[path]
		if !ok {
			conn.sendError(errmsgNoObject, sender, serial)
			conn.handlersLck.RUnlock()
			return
		}
		iface := obj[ifaceName]
		conn.handlersLck.RUnlock()
		m = exportedMethod(iface, name)
	} else {
		conn.handlersLck.RLock()
		if _, ok := conn.handlers[path]; !ok {
			conn.sendError(errmsgNoObject, sender, serial)
			conn.handlersLck.RUnlock()
			return
		}
		for _, v := range conn.handlers[path] {
			m = exportedMethod(v, name)
			if m.IsValid() {
				break
			}
		}
		conn.handlersLck.RUnlock()
	}
	if !m.IsValid() {
		conn.sendError(errmsgUnknownMethod, sender, serial)
		return
	}
	t := m.Type()
	vs := msg.Body
	if t.NumIn() != len(vs) {
		conn.sendError(errmsgInvalidArg, sender, serial)
		return
	}
	pointers := make([]interface{}, len(vs))
	for i := 0; i < t.NumIn(); i++ {
		pointers[i] = reflect.New(t.In(i)).Interface()
	}
	if err := Store(vs, pointers...); err != nil {
		conn.sendError(errmsgInvalidArg, sender, serial)
		return
	}
	params := make([]reflect.Value, len(vs))
	for i := 0; i < len(vs); i++ {
		params[i] = reflect.ValueOf(pointers[i]).Elem()
	}
	ret := m.Call(params)
	out_n := t.NumOut()
	if out_n > 0 && ret[out_n-1].Type() == reflect.TypeOf(&errmsgInvalidArg) {
		if em := ret[out_n-1].Interface().(*Error); em != nil {
			conn.sendError(*em, sender, serial)
			return
		}
		ret = ret[:out_n-1]
	}
	if msg.Flags&FlagNoReplyExpected == 0 {
		reply := new(Message)
		reply.Type = TypeMethodReply
		reply.serial = conn.getSerial()
		reply.Headers = make(map[HeaderField]Variant)
		reply.Headers[FieldDestination] = msg.Headers[FieldSender]
		reply.Headers[FieldReplySerial] = MakeVariant(msg.serial)
		reply.Body = make([]interface{}, len(ret))
		for i := 0; i < len(ret); i++ {
			reply.Body[i] = ret[i].Interface()
		}
		if len(ret) != 0 {
			reply.Headers[FieldSignature] = MakeVariant(SignatureOf(reply.Body...))
		}
		conn.outLck.RLock()
		if !conn.closed {
			conn.out <- reply
		}
		conn.outLck.RUnlock()
	}
}

// Emit emits the given signal on the message bus. The name parameter must be
// formatted as "interface.member", e.g., "org.freedesktop.DBus.NameLost".
func (conn *Conn) Emit(path ObjectPath, name string, values ...interface{}) error {
	if !path.IsValid() {
		return errors.New("dbus: invalid object path")
	}
	i := strings.LastIndex(name, ".")
	if i == -1 {
		return errors.New("dbus: invalid method name")
	}
	iface := name[:i]
	member := name[i+1:]
	if !isValidMember(member) {
		return errors.New("dbus: invalid method name")
	}
	if !isValidInterface(iface) {
		return errors.New("dbus: invalid interface name")
	}
	msg := new(Message)
	msg.Type = TypeSignal
	msg.serial = conn.getSerial()
	msg.Headers = make(map[HeaderField]Variant)
	msg.Headers[FieldInterface] = MakeVariant(iface)
	msg.Headers[FieldMember] = MakeVariant(member)
	msg.Headers[FieldPath] = MakeVariant(path)
	msg.Body = values
	if len(values) > 0 {
		msg.Headers[FieldSignature] = MakeVariant(SignatureOf(values...))
	}
	conn.outLck.RLock()
	defer conn.outLck.RUnlock()
	if conn.closed {
		return ErrClosed
	}
	conn.out <- msg
	return nil
}

// Export registers the given value to be exported as an object on the
// message bus.
//
// If a method call on the given path and interface is received, an exported
// method with the same name is called with v as the receiver if the
// parameters match and the last return value is of type *Error. If this
// *Error is not nil, it is sent back to the caller as an error.
// Otherwise, a method reply is sent with the other return values as its body.
//
// Every method call is executed in a new goroutine, so the method may be called
// in multiple goroutines at once.
//
// Method calls on the interface org.freedesktop.DBus.Peer will be automatically
// handled for every object.
//
// Passing nil as the first parameter will cause conn to cease handling calls on
// the given combination of path and interface.
//
// Export returns an error if path is not a valid path name.
func (conn *Conn) Export(v interface{}, path ObjectPath, iface string) error {
	if !path.IsValid() {
		return errors.New("dbus: invalid path name")
	}
	conn.handlersLck.Lock()
	if v == nil {
		if _, ok := conn.handlers[path]; ok {
			delete(conn.handlers[path], iface)
			if len(conn.handlers[path]) == 0 {
				delete(conn.handlers, path)
			}
		}
		return nil
	}
	if _, ok := conn.handlers[path]; !ok {
		conn.handlers[path] = make(map[string]interface{})
	}
	conn.handlers[path][iface] = v
	conn.handlersLck.Unlock()
	return nil
}

// ReleaseName calls org.freedesktop.DBus.ReleaseName. You should use only this
// method to release a name (see below).
func (conn *Conn) ReleaseName(name string) (ReleaseNameReply, error) {
	var r uint32
	err := conn.busObj.Call("org.freedesktop.DBus.ReleaseName", 0, name).Store(&r)
	if err != nil {
		return 0, err
	}
	if r == uint32(ReleaseNameReplyReleased) {
		conn.namesLck.Lock()
		for i, v := range conn.names {
			if v == name {
				copy(conn.names[i:], conn.names[i+1:])
				conn.names = conn.names[:len(conn.names)-1]
			}
		}
		conn.namesLck.Unlock()
	}
	return ReleaseNameReply(r), nil
}

// RequestName calls org.freedesktop.DBus.RequestName. You should use only this
// method to request a name because package dbus needs to keep track of all
// names that the connection has.
func (conn *Conn) RequestName(name string, flags RequestNameFlags) (RequestNameReply, error) {
	var r uint32
	err := conn.busObj.Call("org.freedesktop.DBus.RequestName", 0, name, flags).Store(&r)
	if err != nil {
		return 0, err
	}
	if r == uint32(RequestNameReplyPrimaryOwner) {
		conn.namesLck.Lock()
		conn.names = append(conn.names, name)
		conn.namesLck.Unlock()
	}
	return RequestNameReply(r), nil
}

func (conn *Conn) Unexport(path ObjectPath, iface string) {
	conn.handlersLck.Lock()
	conn.handlersLck.Unlock()
}

// ReleaseNameReply is the reply to a ReleaseName call.
type ReleaseNameReply uint32

const (
	ReleaseNameReplyReleased ReleaseNameReply = 1 + iota
	ReleaseNameReplyNonExistent
	ReleaseNameReplyNotOwner
)

// RequestNameFlags represents the possible flags for a RequestName call.
type RequestNameFlags uint32

const (
	NameFlagAllowReplacement RequestNameFlags = 1 << iota
	NameFlagReplaceExisting
	NameFlagDoNotQueue
)

// RequestNameReply is the reply to a RequestName call.
type RequestNameReply uint32

const (
	RequestNameReplyPrimaryOwner RequestNameReply = 1 + iota
	RequestNameReplyInQueue
	RequestNameReplyExists
	RequestNameReplyAlreadyOwner
)
