package tool

import (
	"errors"
	"net"
	"net/http"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type RDNS struct{}

func NewRDNS() *RDNS {
	return &RDNS{}
}

func (i *RDNS) Paths() []string {
	return []string{
		"/rdns",
		"/rdns/{host}",
	}
}

func (i *RDNS) Method() string {
	return "GET"
}

func (i *RDNS) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	ip := net.ParseIP(util.GetSourceIPAddress(r))
	hostVar, ok := vars["host"]
	if ok {
		lookupResp, err := net.LookupIP(hostVar)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, errors.New("failed to lookup host")
		}
		if len(lookupResp) == 0 {
			return nil, errors.New("failed to lookup host - no DNS records")
		}
		ip = lookupResp[0]
	}

	rdns, err := net.LookupAddr(ip.String())
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to lookup rDNS")
	}

	if len(rdns) == 0 {
		return nil, errors.New("failed to lookup rDNS - no rDNS records")
	}

	return NewToolResponse(
		NewToolResponseString(rdns[0]),
	), nil
}
