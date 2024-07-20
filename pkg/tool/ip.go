package tool

import (
	"net/http"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
)

type IP struct{}

func NewIP() *IP {
	return &IP{}
}

func (i *IP) Paths() []string {
	return []string{
		"/ip",
	}
}

func (i *IP) Method() string {
	return "GET"
}

func (i *IP) Handle(r *http.Request) (*ToolResponse, error) {
	return NewToolResponse(
		NewToolResponseString(util.GetSourceIPAddress(r)),
	), nil
}
