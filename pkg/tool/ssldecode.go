package tool

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type SSLDecode struct{}

func NewSSLDecode() *SSLDecode {
	return &SSLDecode{}
}

func (s *SSLDecode) Paths() []string {
	return []string{
		"/ssldecode",
	}
}

func (s *SSLDecode) Method() string {
	return "POST"
}

func (s *SSLDecode) Handle(r *http.Request) (*ToolResponse, error) {
	if r.Body == nil {
		return nil, errors.New("missing SSL certificate(s)")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to read SSL certificate(s)")
	}
	r.Body.Close()

	der, _ := pem.Decode(body)
	if der == nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to decode SSL certificate(s)")
	}

	cert, err := x509.ParseCertificate(der.Bytes)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to parse SSL certificate(s)")
	}

	var sans []string
	if cert.DNSNames != nil {
		sans = append(sans, cert.DNSNames...)
	}
	if cert.IPAddresses != nil {
		for _, ipAddress := range cert.IPAddresses {
			sans = append(sans, ipAddress.String())
		}
	}

	return NewToolResponse(
		&SSLDecodeResponseData{
			CommonName:         cert.Subject.CommonName,
			SANs:               sans,
			Organisation:       cert.Subject.Organization,
			City:               cert.Subject.Locality,
			Country:            cert.Subject.Country,
			Serial:             cert.Subject.SerialNumber,
			ValidFrom:          cert.NotBefore.String(),
			ValidTo:            cert.NotAfter.String(),
			IssuerName:         cert.Issuer.CommonName,
			IssuerOrganisation: cert.Issuer.Organization,
			IssuerCity:         cert.Issuer.Locality,
			IssuerCountry:      cert.Issuer.Country,
			IssuerSerial:       cert.Issuer.SerialNumber,
		},
	), nil
}

type SSLDecodeResponseData struct {
	CommonName         string   `json:"common_name"`
	SANs               []string `json:"sans"`
	Organisation       []string `json:"organisation"`
	City               []string `json:"city"`
	Country            []string `json:"country"`
	Serial             string   `json:"serial"`
	ValidFrom          string   `json:"valid_from"`
	ValidTo            string   `json:"valid_to"`
	IssuerName         string   `json:"issuer_name"`
	IssuerOrganisation []string `json:"issuer_organisation"`
	IssuerCity         []string `json:"issuer_city"`
	IssuerCountry      []string `json:"issuer_country"`
	IssuerSerial       string   `json:"issuer_serial"`
}

func (r *SSLDecodeResponseData) String() string {
	return fmt.Sprintf(`> Certificate
Common Name:           %s
SANs:                  %s
Organisation:          %s
City:                  %s
Country:               %s
Serial:                %s
Valid From:            %s
Valid To:              %s

> Issuer
Issuer Name:           %s
Issuer Organisation:   %s
Issuer City:           %s
Issuer Country:        %s
Issuer Serial:         %s`, r.CommonName, strings.Join(r.SANs, ", "), strings.Join(r.Organisation, ", "), strings.Join(r.City, ", "), strings.Join(r.Country, ", "), r.Serial, r.ValidFrom, r.ValidTo, r.IssuerName, strings.Join(r.IssuerOrganisation, ", "), strings.Join(r.IssuerCity, ", "), strings.Join(r.IssuerCountry, ", "), r.IssuerSerial)
}
