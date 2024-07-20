package tool

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type Keypair struct{}

func NewKeypair() *Keypair {
	return &Keypair{}
}

func (k *Keypair) Paths() []string {
	return []string{
		"/keypair",
	}
}

func (k *Keypair) Method() string {
	return "GET"
}

func (k *Keypair) Handle(r *http.Request) (*ToolResponse, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to generate private key")
	}

	err = privateKey.Validate()
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to validate private key")
	}

	privBuf := new(bytes.Buffer)
	err = pem.Encode(privBuf, &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to encode private key")
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to create public key")
	}

	return NewToolResponse(
		&KeypairResponseData{
			PrivateKey: strings.TrimSpace(privBuf.String()),
			PublicKey:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))),
		},
	), nil
}

type KeypairResponseData struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func (r *KeypairResponseData) String() string {
	return fmt.Sprintf("%s\n\n%s", r.PrivateKey, r.PublicKey)
}
