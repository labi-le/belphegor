// todo put in a separate package

package internal

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"io"
	"math"
	"sync"
)

const (
	BitSize = 1024
)

type Cipher struct {
	private crypto.PrivateKey
	public  crypto.PublicKey

	size int
}

func NewCipher() *Cipher {
	privateKey, cipherErr := rsa.GenerateKey(rand.Reader, BitSize)
	if cipherErr != nil {
		log.Fatal().Msgf("failed to generate private key: %s", cipherErr)
	}

	return &Cipher{
		private: privateKey,
		public:  privateKey.Public(),
		size:    encryptSize(privateKey.Public()),
	}
}

func (c *Cipher) PublicKeyBytes() []byte {
	return publicKey2Bytes(c.public)
}

func (c *Cipher) Encrypt(src []byte) (*types.EncryptedMessage, error) {
	var (
		enc types.EncryptedMessage
	)

	if len(src) <= c.size {
		byts, err := rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			c.public.(*rsa.PublicKey),
			src,
			nil,
		)

		enc.Parts = [][]byte{byts}
		return &enc, err
	}

	var (
		parts = explodeBySize(src, c.size)
	)

	parallelEnc(&enc, parts, c)
	return &enc, nil
}

func nonParallelEnc(t *types.EncryptedMessage, parts [][]byte, c *Cipher) {
	for _, part := range parts {
		encByt, encErr := rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			c.public.(*rsa.PublicKey),
			part,
			nil,
		)
		if encErr != nil {
			panic(encErr)
		}

		t.Parts = append(t.Parts, encByt)
	}
}

func parallelEnc(enc *types.EncryptedMessage, parts [][]byte, c *Cipher) {
	var (
		encChan = make(chan encPart)
		wg      sync.WaitGroup
	)

	enc.Parts = make([][]byte, len(parts))
	go func() {
		for part := range encChan {
			enc.Parts[part.index] = part.byt
		}
	}()

	for i, part := range parts {
		wg.Add(1)

		part := part
		i := i
		go func() {
			defer wg.Done()

			encByt, encErr := rsa.EncryptOAEP(
				sha256.New(),
				rand.Reader,
				c.public.(*rsa.PublicKey),
				part,
				nil,
			)
			if encErr != nil {
				panic(encErr)
			}

			encChan <- encPart{
				index: i,
				byt:   encByt,
			}
		}()
	}

	wg.Wait()
}

type encPart struct {
	index int
	byt   []byte
}

func (c *Cipher) EncryptMessage(msg proto.Message) (*types.EncryptedMessage, error) {
	return c.Encrypt(encode(msg))
}

func (c *Cipher) EncryptWriter(msg proto.Message, w io.Writer) (int, error) {
	message, err := c.EncryptMessage(msg)
	if err != nil {
		return 0, err
	}
	return encodeWriter(message, w)
}

func (c *Cipher) Decrypt(src *types.EncryptedMessage) ([]byte, error) {
	if len(src.Parts) == 1 {
		return rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			c.private.(*rsa.PrivateKey),
			src.Parts[0],
			nil,
		)
	}

	var (
		buf = new(bytes.Buffer)
	)
	for _, part := range src.Parts {
		dec, decErr := rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			c.private.(*rsa.PrivateKey),
			part,
			nil,
		)
		if decErr != nil {
			return nil, decErr
		}

		buf.Write(dec)
	}

	return buf.Bytes(), nil
}

func (c *Cipher) DecryptReader(r io.Reader, dst proto.Message) error {
	var encrypt types.EncryptedMessage
	if decodeEnc := decodeReader(r, &encrypt); decodeEnc != nil {
		return decodeEnc
	}

	decrypt, decErr := c.Decrypt(&encrypt)
	if decErr != nil {
		return decErr
	}

	return proto.Unmarshal(decrypt, dst)
}

func encryptSize(pub crypto.PublicKey) int {
	return pub.(*rsa.PublicKey).Size() - 2*sha256.New().Size() - 2
}

func explodeBySize(src []byte, size int) [][]byte {
	numParts := int(math.Ceil(float64(len(src)) / float64(size)))
	parts := make([][]byte, numParts)

	for i := 0; i < numParts; i++ {
		start := i * size
		end := (i + 1) * size
		if end > len(src) {
			end = len(src)
		}
		parts[i] = byteslice.Get(end - start)
		copy(parts[i], src[start:end])
	}

	return parts
}

func publicKey2Bytes(publicKey crypto.PublicKey) []byte {
	publicKeyBytes, marshalErr := x509.MarshalPKIXPublicKey(publicKey)
	if marshalErr != nil {
		log.Fatal().Msgf("failed to marshal public key: %s", marshalErr)
	}
	return publicKeyBytes
}

func bytes2PublicKey(publicKeyBytes []byte) crypto.PublicKey {
	publicKey, parseErr := x509.ParsePKIXPublicKey(publicKeyBytes)
	if parseErr != nil {
		log.Fatal().Msgf("failed to parse public key: %s", parseErr)
	}
	return publicKey
}
