package tool

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type SSL struct{}

func NewSSL() *SSL {
	return &SSL{}
}

func (i *SSL) Paths() []string {
	return []string{
		"/ssl",
		"/ssl/{host}",
		"/ssl/{host}/{port}",
	}
}

func (i *SSL) Method() string {
	return "GET"
}

func (i *SSL) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	host, ok := vars["host"]
	if !ok {
		host = util.GetSourceIPAddress(r)
	}

	port := 443
	portVar, ok := vars["port"]

	var err error
	if ok {
		port, err = strconv.Atoi(portVar)
		if err != nil || (port < 1 || port > 65535) {
			return nil, errors.New("port must be an integer between 1 and 65535")
		}
	}

	dialer := &tls.Dialer{
		Config: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to connect to TLS host")
	}
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	certs := tlsConn.ConnectionState().PeerCertificates

	interPool := x509.NewCertPool()
	var responseChain []SSLChainResponseData
	for i, cert := range certs {
		responseChainCert := SSLChainResponseData{
			CommonName:         cert.Subject.CommonName,
			IssuerCommonName:   cert.Issuer.CommonName,
			IssuerOrganisation: strings.Join(cert.Issuer.Organization, ", "),
		}

		if i > 0 {
			responseChainCert.ValidIssuer = string(certs[i-1].AuthorityKeyId) == string(cert.SubjectKeyId)
		}

		interPool.AddCert(cert)

		responseChain = append(responseChain, responseChainCert)
	}

	_, verifyErr := certs[0].Verify(x509.VerifyOptions{
		Intermediates: interPool,
		DNSName:       host,
	})

	var verifyErrString string
	if verifyErr != nil {
		verifyErrString = verifyErr.Error()
	}

	return NewToolResponse(&SSLResponseData{
		Chain: responseChain,
		Valid: verifyErr == nil,
		Error: verifyErrString,
	}), nil
}

type SSLResponseData struct {
	Chain []SSLChainResponseData `json:"chain"`
	Valid bool                   `json:"valid"`
	Error string                 `json:"error,omitempty"`
}

type SSLChainResponseData struct {
	CommonName         string `json:"common_name"`
	IssuerCommonName   string `json:"issuer_common_name"`
	IssuerOrganisation string `json:"issuer_organisation"`
	ValidIssuer        bool   `json:"valid_issuer"`
}

func (r *SSLResponseData) String() string {
	str := fmt.Sprintf("Valid:    %t\n", r.Valid)
	if !r.Valid {
		str = str + fmt.Sprintf("Error:    %s\n", r.Error)
	}
	str = str + "\n"

	for i, cert := range r.Chain {
		if i > 0 {
			padding := strings.Repeat("       ", i-1)
			validChar := "✖"
			if cert.ValidIssuer {
				validChar = "✔"
			}
			str = str + fmt.Sprintf(`%[1]s│
%[1]s└─[%[2]s]─ `, padding, validChar)
		}

		str = str + fmt.Sprintf("%s (Issuer: %s)\n", cert.CommonName, cert.IssuerCommonName)
	}

	return str
}
