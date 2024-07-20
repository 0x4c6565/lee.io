package tool

import (
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"crypto/rand"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Password struct{}

func NewPassword() *Password {
	return &Password{}
}

func (p *Password) Paths() []string {
	return []string{
		"/pw",
		"/pw/",
		"/pw/{length}",
		"/password",
		"/password/",
		"/password/{length}",
	}
}

func (p *Password) Method() string {
	return "GET"
}

func (p *Password) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)
	noSymbols := r.URL.Query().Has("nosymbols")

	length := 12
	var err error
	lengthVar, ok := vars["length"]
	if ok {
		length, err = strconv.Atoi(lengthVar)
		if err != nil || length < 4 || length > 256 {
			return nil, errors.New("invalid length")
		}
	}

	password, err := generatePassword(length, noSymbols)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to generate password")
	}

	return NewToolResponse(
		NewToolResponseString(password),
	), nil
}

func generatePassword(length int, noSymbols bool) (string, error) {
	lettersUpperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lettersLowerChars := "abcdefghijklmnopqrstuvwxyz"
	numberChars := "0123456789"
	specialChars := "!@#$%^&*()_+-=[]{}\\|;':\",.<>/?`~"
	charSet := lettersUpperChars + lettersLowerChars + numberChars
	if !noSymbols {
		charSet = charSet + specialChars
	}

	for {
		ret := make([]byte, length)
		for i := 0; i < length; i++ {
			num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
			if err != nil {
				return "", err
			}
			ret[i] = charSet[num.Int64()]
		}

		if !strings.ContainsAny(string(ret), lettersUpperChars) {
			continue
		}

		if !strings.ContainsAny(string(ret), lettersLowerChars) {
			continue
		}

		if !strings.ContainsAny(string(ret), numberChars) {
			continue
		}

		if !noSymbols && !strings.ContainsAny(string(ret), specialChars) {
			continue
		}

		return string(ret), nil
	}
}
