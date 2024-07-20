package tool

import (
	"errors"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

type EUI64 struct{}

func NewEUI64() *EUI64 {
	return &EUI64{}
}

func (i *EUI64) Paths() []string {
	return []string{
		"/eui64",
		"/eui64/",
		"/eui64/{prefix}",
		"/eui64/{prefix}/",
		"/eui64/{prefix}/{mac}",
	}
}

func (i *EUI64) Method() string {
	return "GET"
}

func (i *EUI64) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)
	prefix, ok := vars["prefix"]
	if !ok {
		return nil, errors.New("missing prefix")
	}

	mac, ok := vars["mac"]
	if !ok {
		return nil, errors.New("missing mac")
	}

	parsedPrefix := net.ParseIP(prefix)
	if parsedPrefix == nil || parsedPrefix.To16() == nil {
		return nil, errors.New("invalid prefix")
	}

	if !allZeroes(parsedPrefix[8:16]) {
		return nil, errors.New("invalid prefix - must be < 64 bits")
	}

	parsedMac, err := net.ParseMAC(mac)
	if err != nil {
		return nil, errors.New("invalid mac address")
	}

	if len(parsedMac) != 6 && len(parsedMac) != 8 {
		return nil, errors.New("invalid mac - must be in EUI-48 or EUI-64 form")
	}

	eui64 := generateEUI(parsedPrefix, parsedMac)

	return NewToolResponse(
		NewToolResponseString(eui64.String()),
	), nil
}

func generateEUI(prefix net.IP, mac net.HardwareAddr) net.IP {
	ip := make(net.IP, 16)
	copy(ip[0:8], prefix[0:8])

	// MAC in EUI64 form
	if len(mac) == 8 {
		copy(ip[8:16], mac)
		ip[8] ^= 0x02
		return ip
	}

	// MAC in EUI48 form
	copy(ip[8:11], mac[0:3])
	ip[8] ^= 0x02
	ip[11] = 0xff
	ip[12] = 0xfe
	copy(ip[13:16], mac[3:6])

	return ip
}

func allZeroes(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return false
		}
	}

	return true
}
