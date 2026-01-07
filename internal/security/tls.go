package security

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/rs/zerolog"
)

func MakeTLSConfig(secret string, logger zerolog.Logger) (*tls.Config, error) {
	//nolint:mnd,gosec //shut up
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("rand.Int: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour * 365 * 10),
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"), // localhost
			net.ParseIP("0.0.0.0"),   // any
			net.IPv4(192, 168, 0, 0), // 192.168.0.0/16
			net.IPv4(10, 0, 0, 0),    // 10.0.0.0/8
			net.IPv4(172, 16, 0, 0),  // 172.16.0.0/12
		},
	}

	privateKey, publicKey, err2 := genKey(secret)
	if err2 != nil {
		return nil, err2
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("x509.CreateCertificate: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalPKCS8PrivateKey: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls.X509KeyPair: %w", err)
	}

	conf := &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"belphegor"},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}

	populateKeyLog(logger, conf)

	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return ErrPeerSecretMissing
		}

		peerCert, pErr := x509.ParseCertificate(rawCerts[0])
		if pErr != nil {
			return fmt.Errorf("failed to parse peer cert: %w", pErr)
		}

		myCert, mErr := x509.ParseCertificate(certDER)
		if mErr != nil {
			return fmt.Errorf("failed to parse my cert: %w", mErr)
		}

		myPub, myOk := myCert.PublicKey.(ed25519.PublicKey)
		peerPub, peerOk := peerCert.PublicKey.(ed25519.PublicKey)

		if myOk != peerOk {
			if myOk {
				return ErrPeerSecretMissing
			}
			return ErrLocalSecretMissing
		}

		if !myOk && !peerOk {
			return nil
		}

		if !bytes.Equal(myPub, peerPub) {
			return ErrSecretMismatch
		}

		return nil
	}

	return conf, nil
}

func genKey(secret string) (crypto.PrivateKey, crypto.PublicKey, error) {
	if secret != "" {
		seed := sha256.Sum256([]byte(secret))
		pk := ed25519.NewKeyFromSeed(seed[:])
		return pk, pk.Public(), nil
	}

	ecdsaPriv, eErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if eErr != nil {
		return nil, nil, fmt.Errorf("ecdsa.GenerateKey: %w", eErr)
	}
	return ecdsaPriv, ecdsaPriv.Public(), nil
}
