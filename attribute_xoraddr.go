package stun

import (
	"errors"
	"fmt"
	"net"
)

const (
	familyIPv4 byte = 0x01
	familyIPv6 byte = 0x02
)

// XORMappedAddress implements XOR-MAPPED-ADDRESS attribute.
type XORMappedAddress struct {
	IP   net.IP
	Port int
}

func (a XORMappedAddress) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

// Is p all zeros?
func isZeros(p net.IP) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}

// ErrBadIPLength means that len(IP) is not net.{IPv6len,IPv4len}.
var ErrBadIPLength = errors.New("invalid length of IP value")

// AddTo adds XOR-MAPPED-ADDRESS to m. Can return ErrBadIPLength
// if len(a.IP) is invalid.
func (a *XORMappedAddress) AddTo(m *Message) error {
	var (
		family = familyIPv4
		ip     = a.IP
	)
	if len(a.IP) == net.IPv6len {
		// Optimized for performance. See net.IP.To4 method.
		if isZeros(ip[0:10]) && ip[10] == 0xff && ip[11] == 0xff {
			ip = ip[12:16] // like in ip.To4()
		} else {
			family = familyIPv6
		}
	} else if len(ip) != net.IPv4len {
		return ErrBadIPLength
	}
	value := make([]byte, 32+128)
	value[0] = 0 // first 8 bits are zeroes
	xorValue := make([]byte, net.IPv6len)
	copy(xorValue[4:], m.TransactionID[:])
	bin.PutUint32(xorValue[0:4], magicCookie)
	bin.PutUint16(value[0:2], uint16(family))
	bin.PutUint16(value[2:4], uint16(a.Port^magicCookie>>16))
	xorBytes(value[4:4+len(ip)], ip, xorValue)
	m.Add(AttrXORMappedAddress, value[:4+len(ip)])
	return nil
}

// GetFrom decodes XOR-MAPPED-ADDRESS attribute in message and returns
// error if any. While decoding, a.IP is reused if possible and can be
// rendered to invalid state (e.g. if a.IP was set to IPv6 and then
// IPv4 value were decoded into it), be careful.
//
// Example:
//
// expectedIP := net.ParseIP("213.141.156.236")
// expectedIP.String() // 213.141.156.236, 16 bytes, first 12 of them are zeroes
// expectedPort := 21254
// addr := &XORMappedAddress{
//     IP:   expectedIP,
//     Port: expectedPort,
// }
// // addr were added to message that is decoded as newMessage
// // ...
//
// addr.GetFrom(newMessage)
// addr.IP.String()    // 213.141.156.236, net.IPv4Len
// expectedIP.String() // d58d:9cec::ffff:d58d:9cec, 16 bytes, first 4 are IPv4
// // now we have len(expectedIP) = 16 and len(addr.IP) = 4.
func (a *XORMappedAddress) GetFrom(m *Message) error {
	v, err := m.Get(AttrXORMappedAddress)
	if err != nil {
		return err
	}
	family := byte(bin.Uint16(v[0:2]))
	if family != familyIPv6 && family != familyIPv4 {
		return newDecodeErr("xor-mapped address", "family",
			fmt.Sprintf("bad value %d", family),
		)
	}
	ipLen := net.IPv4len
	if family == familyIPv6 {
		ipLen = net.IPv6len
	}
	// Ensuring len(a.IP) == ipLen and reusing a.IP.
	if len(a.IP) < ipLen {
		a.IP = a.IP[:cap(a.IP)]
		for len(a.IP) < ipLen {
			a.IP = append(a.IP, 0)
		}
	}
	a.IP = a.IP[:ipLen]
	for i := range a.IP {
		a.IP[i] = 0
	}
	a.Port = int(bin.Uint16(v[2:4])) ^ (magicCookie >> 16)
	xorValue := make([]byte, 4+transactionIDSize)
	bin.PutUint32(xorValue[0:4], magicCookie)
	copy(xorValue[4:], m.TransactionID[:])
	xorBytes(a.IP, v[4:], xorValue)
	return nil
}