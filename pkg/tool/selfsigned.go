package tool

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type SelfSigned struct{}

func NewSelfSigned() *SelfSigned {
	return &SelfSigned{}
}

func (s *SelfSigned) Paths() []string {
	return []string{
		"/selfsigned",
		"/selfsigned/{hosts}",
		"/selfsigned/{hosts}/{days}",
	}
}

func (s *SelfSigned) Method() string {
	return "GET"
}

func (s *SelfSigned) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	hosts, ok := vars["hosts"]
	if !ok {
		return nil, errors.New("missing hosts")
	}

	hostsSplit := strings.Split(hosts, ",")

	daysInt := 365

	days, ok := vars["days"]
	if ok {
		daysInt, err := strconv.Atoi(days)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, errors.New("failed to parse days")
		}

		if daysInt < 1 || daysInt > 10000 {
			return nil, errors.New("invalid days")
		}
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to generate private key")
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour * time.Duration(daysInt))
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to generate serial number")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   hostsSplit[0],
			Organization: []string{"Default Company Ltd"},
		},
		NotBefore:             time.Now(),
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hostsSplit {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}

	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to create certificate")
	}

	certBuf := new(bytes.Buffer)
	err = pem.Encode(certBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to encode certificate")
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("unable to marshal private key")
	}

	privBuf := new(bytes.Buffer)
	err = pem.Encode(privBuf, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to encode private key")
	}

	return NewToolResponse(&SelfSignedResponseData{
		Cert: strings.TrimSpace(certBuf.String()),
		Key:  strings.TrimSpace(privBuf.String()),
	}), nil
}

type SelfSignedResponseData struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

func (r *SelfSignedResponseData) String() string {
	return fmt.Sprintf("%s\n%s", r.Cert, r.Key)
}
