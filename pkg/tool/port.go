package tool

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/gorilla/mux"
)

type Port struct{}

func NewPort() *Port {
	return &Port{}
}

func (p *Port) Paths() []string {
	return []string{
		"/port",
		"/port/",
		"/port/{port}",
		"/port/{port}/",
		"/port/{port}/{host}",
	}
}

func (p *Port) Method() string {
	return "GET"
}

func (p *Port) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	port, ok := vars["port"]
	if !ok {
		return nil, errors.New("missing port")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil || (portInt < 1 || portInt > 65535) {
		return nil, errors.New("port must be an integer between 1 and 65535")
	}

	host, ok := vars["host"]
	if !ok {
		host = util.GetSourceIPAddress(r)
	}

	if strings.Contains(host, ":") {
		host = fmt.Sprintf("[%s]", host)
	}

	status := "Open"
	_, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, portInt))
	if err != nil {
		status = "Closed"
	}

	return NewToolResponse(&PortResponseData{
		Host:   host,
		HostIP: "",
		Status: status,
	}), nil
}

type PortResponseData struct {
	Host   string `json:"host"`
	HostIP string `json:"host_ip"`
	Status string `json:"status"`
}

func (r *PortResponseData) String() string {
	return fmt.Sprintf("Host: %s\nHost IP: %s\nStatus: %s", r.Host, r.HostIP, r.Status)
}
