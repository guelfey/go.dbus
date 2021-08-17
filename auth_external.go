package dbus

import (
)

// AuthExternal returns an Auth that authenticates as the given user with the
// EXTERNAL mechanism.
func AuthExternal(user string) Auth {
	return authExternal{user}
}

// AuthExternal implements the EXTERNAL authentication mechanism.
type authExternal struct {
	user string
}

func (a authExternal) FirstData() ([]byte, []byte, AuthStatus) {
	//b := make([]byte, 2*len(a.user))
	//hex.Encode(b, []byte(a.user))
	return []byte("EXTERNAL"), nil, AuthContinue
}

func (a authExternal) HandleData(data [][]byte) ([][]byte, AuthStatus) {
	if (len(data) != 0) {
		return nil, AuthError
	}
	return nil, AuthOk
}
