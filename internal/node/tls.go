package node

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/labi-le/belphegor/pkg/ip"
	"github.com/rs/zerolog/log"
	"math/big"
	"time"
)

func generateTLSConfig(opts EncryptionOptions) *tls.Config {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	ips, _ := ip.GetLocalIPs()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  ips,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Создаем пару ключ-сертификат
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// Настраиваем конфигурацию TLS
	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"belphegor"},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: !opts.Enable,
	}
}
