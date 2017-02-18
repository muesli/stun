package stun

import "errors"

// NewUsername returns Username with provided value.
func NewUsername(username string) Username {
	return Username(username)
}

// Username represents USERNAME attribute.
//
// https://tools.ietf.org/html/rfc5389#section-15.3
type Username []byte

func (u Username) String() string {
	return string(u)
}

const maxUsernameB = 513

// ErrUsernameTooBig means that USERNAME value is bigger that 513 bytes.
var ErrUsernameTooBig = errors.New("USERNAME value bigger than 513 bytes")

// AddTo adds USERNAME attribute to message.
func (u Username) AddTo(m *Message) error {
	if len(u) > maxUsernameB {
		return ErrUsernameTooBig
	}
	m.Add(AttrUsername, u)
	return nil
}

// GetFrom gets USERNAME from message.
func (u *Username) GetFrom(m *Message) error {
	v, err := m.Get(AttrUsername)
	if err != nil {
		return err
	}
	*u = v
	return nil
}