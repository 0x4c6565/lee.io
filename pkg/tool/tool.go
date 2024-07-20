package tool

import (
	"net/http"
	"strings"
)

type Tool interface {
	Paths() []string
	Method() string
	Handle(r *http.Request) (*ToolResponse, error)
}

type ToolResponseData interface {
	String() string
}

type ToolResponseString string

func NewToolResponseString(s string) *ToolResponseString {
	rs := ToolResponseString(s)
	return &rs
}

func (r *ToolResponseString) String() string {
	return string(*r)
}

type ToolResponseStringSlice []string

func NewToolResponseStringSlice(s []string) ToolResponseStringSlice {
	return ToolResponseStringSlice(s)
}

func (r ToolResponseStringSlice) String() string {
	return strings.Join(r, "\n")
}

type ToolResponse struct {
	Data ToolResponseData
}

func NewToolResponse(d ToolResponseData) *ToolResponse {
	return &ToolResponse{
		Data: d,
	}
}

type CronSpec struct {
	Cron string
	Func func()
}

type ToolCron interface {
	Cron() CronSpec
}
