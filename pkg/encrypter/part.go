package encrypter

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"io"
	"math"
	"sync"
)

type Cipher struct {
	private crypto.PrivateKey
	public  crypto.PublicKey

	size int
}

func (c *Cipher) Public() crypto.PublicKey {
	return c.public
}

func NewCipher(privKey crypto.PrivateKey, pubKey crypto.PublicKey) *Cipher {
	return &Cipher{
		private: privKey,
		public:  pubKey,
		size:    encryptSize(pubKey),
	}
}

func (c *Cipher) PublicKeyBytes() []byte {
	return PublicKey2Bytes(c.Public())
}

func (c *Cipher) Sign(rand io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	if len(digest) <= c.size {
		//return signer.(crypto.Signer).Sign(rand, digest, opts)
		return rsa.EncryptOAEP(
			sha256.New(),
			rand,
			c.public.(*rsa.PublicKey),
			digest,
			nil,
		)

	}

	return parallelEnc(c, rand, explodeBySize(digest, c.size))
}

func (c *Cipher) Decrypt(rand io.Reader, msg []byte, _ crypto.DecrypterOpts) (plaintext []byte, err error) {
	return c.decrypt(rand, msg)
}

func (c *Cipher) decrypt(rand io.Reader, msg []byte) ([]byte, error) {
	ks := c.public.(*rsa.PublicKey).Size()
	parts := explodeBySize(msg, ks)

	var wg sync.WaitGroup
	decChan := make(chan part, len(parts))

	for i, portion := range parts {
		wg.Add(1)
		i := i
		portion := portion
		go func() {
			defer wg.Done()
			dec, err := rsa.DecryptOAEP(
				sha256.New(),
				rand,
				c.private.(*rsa.PrivateKey),
				portion,
				nil,
			)
			decChan <- part{index: i, byt: dec, error: err}
		}()
	}

	wg.Wait()
	close(decChan)

	// Собираем дешифрованные части в порядке их индекса
	decryptedParts := make([][]byte, len(parts))
	for decPart := range decChan {
		if decPart.error != nil {
			return nil, decPart.error
		}
		decryptedParts[decPart.index] = decPart.byt
	}

	var result bytes.Buffer
	for _, decrypted := range decryptedParts {
		result.Write(decrypted)
	}

	return result.Bytes(), nil
}

func parallelEnc(enc *Cipher, rand io.Reader, digest [][]byte) (byt []byte, err error) {
	var (
		encChan = make(chan part)
		data    bytes.Buffer
	)

	var readWg sync.WaitGroup
	readWg.Add(1)
	go func() {
		parts := make([][]byte, len(digest))

		defer readWg.Done()
		for part := range encChan {
			if part.error != nil {
				err = part.error
				return
			}
			parts[part.index] = part.byt
		}

		for _, part := range parts {
			data.Write(part)
		}
	}()

	var wg sync.WaitGroup
	for i, portion := range digest {
		wg.Add(1)

		portion := portion
		i := i
		go func() {
			defer wg.Done()

			encByt, encErr := rsa.EncryptOAEP(
				sha256.New(),
				rand,
				enc.public.(*rsa.PublicKey),
				portion,
				nil,
			)
			//encByt, encErr := enc.private.(crypto.Signer).Sign(rand, portion, opts)

			encChan <- part{
				index: i,
				byt:   encByt,
				error: encErr,
			}
		}()
	}

	wg.Wait()

	close(encChan)
	readWg.Wait()

	return data.Bytes(), nil
}

type part struct {
	error
	index int
	byt   []byte
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

func PublicKey2Bytes(publicKey crypto.PublicKey) []byte {
	publicKeyBytes, marshalErr := x509.MarshalPKIXPublicKey(publicKey)
	if marshalErr != nil {
		log.Fatal().Msgf("failed to marshal public key: %s", marshalErr)
	}
	return publicKeyBytes
}

func Bytes2PublicKey(publicKeyBytes []byte) crypto.PublicKey {
	publicKey, parseErr := x509.ParsePKIXPublicKey(publicKeyBytes)
	if parseErr != nil {
		log.Fatal().Msgf("failed to parse public key: %s", parseErr)
	}
	return publicKey
}
