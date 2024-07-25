package tool

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type UUID struct{}

func NewUUID() *UUID {
	return &UUID{}
}

func (u *UUID) Paths() []string {
	return []string{
		"/uuid",
		"/uuid/{count}",
	}
}

func (u *UUID) Method() string {
	return "GET"
}

func (u *UUID) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)
	nilUUID := r.URL.Query().Has("nil") || r.URL.Query().Has("null") || r.URL.Query().Has("empty")

	count := 1
	countVar, ok := vars["count"]
	var err error
	if ok {
		count, err = strconv.Atoi(countVar)
		if err != nil {
			return nil, errors.New("invalid count")
		}

		if count < 1 {
			count = 1
		}
		if count > 100 {
			count = 100
		}
	}

	var uuids []string
	for i := 0; i < count; i++ {
		if nilUUID {
			uuids = append(uuids, "00000000-0000-0000-0000-000000000000")
		} else {
			u, err := uuid.NewRandom()
			if err != nil {
				log.Error().Err(err).Send()
				return nil, errors.New("failed to generate UUID")
			}

			uuids = append(uuids, u.String())
		}
	}

	return NewToolResponse(
		NewToolResponseStringSlice(uuids),
	), nil
}
