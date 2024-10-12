package tool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Whois struct{}

func NewWhois() *Whois {
	return &Whois{}
}

func (w *Whois) Paths() []string {
	return []string{
		"/whois",
		"/whois/{host}",
	}
}

func (w *Whois) Method() string {
	return "GET"
}

func (w *Whois) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	host, ok := vars["host"]
	if !ok {
		host = util.GetSourceIPAddress(r)
	}

	whoisServer, err := w.doQuery("whois.iana.org", host)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to query whois server")
	}

	defer whoisServer.Close()

	parsedWhoisServer, err := w.parseWhoisServer(whoisServer)
	if err != nil {
		return nil, err
	}

	targetWhoisServer, err := w.doQuery(parsedWhoisServer, host)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to query whois server")
	}

	defer targetWhoisServer.Close()

	response, err := io.ReadAll(targetWhoisServer)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to read whois response")
	}

	return NewToolResponse(
		NewToolResponseString(fmt.Sprintf("Whois server: %s\n\n%s", parsedWhoisServer, response)),
	), nil
}

func (w *Whois) parseWhoisServer(response io.Reader) (string, error) {
	scanner := bufio.NewScanner(response)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "whois:") {
			return strings.TrimSpace(strings.Replace(string(scanner.Text()), "whois:", "", 1)), nil
		}
	}

	return "", errors.New("no whois server found")
}

func (w *Whois) doQuery(server, query string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:43", server), time.Second*10)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write([]byte(fmt.Sprintf("%s\r\n", query)))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
