package encrypter

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"github.com/labi-le/belphegor/pkg/pool/byteslice"
	"github.com/rs/zerolog/log"
	"io"
	"math"
)

type Cipher struct {
	private crypto.PrivateKey
	public  crypto.PublicKey
}

func (c *Cipher) Public() crypto.PublicKey {
	return c.public
}

func NewCipher(privKey crypto.PrivateKey, pubKey crypto.PublicKey) *Cipher {
	return &Cipher{
		private: privKey,
		public:  pubKey,
	}
}

type EncryptedMessage struct {
	KeyLength  uint32 // Length of the encrypted AES key
	Key        []byte // AES encrypted key
	Nonce      []byte // GCM nonce
	CipherText []byte // Encrypted data
}

func (m *EncryptedMessage) ToBytes() []byte {
	keyLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(keyLenBytes, m.KeyLength)

	totalLen := 4 + len(m.Key) + len(m.Nonce) + len(m.CipherText)
	result := make([]byte, totalLen)

	offset := 0
	copy(result[offset:], keyLenBytes)
	offset += 4

	copy(result[offset:], m.Key)
	offset += len(m.Key)

	copy(result[offset:], m.Nonce)
	offset += len(m.Nonce)

	copy(result[offset:], m.CipherText)

	return result
}

func ParseEncryptedMessage(data []byte) (*EncryptedMessage, error) {
	if len(data) < 4 {
		return nil, errors.New("data too short")
	}

	msg := &EncryptedMessage{}
	msg.KeyLength = binary.BigEndian.Uint32(data[:4])
	offset := 4

	if len(data) < offset+int(msg.KeyLength) {
		return nil, errors.New("invalid key length")
	}

	msg.Key = data[offset : offset+int(msg.KeyLength)]
	offset += int(msg.KeyLength)

	if len(data) < offset+12 { // 12 - стандартный размер nonce для GCM
		return nil, errors.New("data too short for nonce")
	}

	msg.Nonce = data[offset : offset+12]
	offset += 12

	msg.CipherText = data[offset:]

	return msg, nil
}

func (c *Cipher) Sign(rand io.Reader, data []byte, _ crypto.SignerOpts) ([]byte, error) {
	aesKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand, aesKey); err != nil {
		return nil, err
	}

	encryptedKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand,
		c.public.(*rsa.PublicKey),
		aesKey,
		nil,
	)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	msg := &EncryptedMessage{
		KeyLength:  uint32(len(encryptedKey)),
		Key:        encryptedKey,
		Nonce:      nonce,
		CipherText: ciphertext,
	}

	return msg.ToBytes(), nil
}

func (c *Cipher) Decrypt(rand io.Reader, msg []byte, _ crypto.DecrypterOpts) (plaintext []byte, err error) {
	encMsg, err := ParseEncryptedMessage(msg)
	if err != nil {
		return nil, err
	}

	aesKey, err := rsa.DecryptOAEP(
		sha256.New(),
		rand,
		c.private.(*rsa.PrivateKey),
		encMsg.Key,
		nil,
	)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, encMsg.Nonce, encMsg.CipherText, nil)
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
