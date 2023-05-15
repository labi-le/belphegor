package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var ErrDataIsNotLongEnough = errors.New("encrypted data is not long enough")

type Cipher struct {
	Key []byte
}

func NewEncryption(key []byte) *Cipher {
	return &Cipher{
		Key: key,
	}
}

func (e *Cipher) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}

	// Generate random initialization vector
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Create a CBC Encryption Mode Using the AES Block Cipher
	mode := cipher.NewCBCEncrypter(block, iv)

	// Add an initialization vector to the encrypted data (at the beginning)
	encrypted := make([]byte, len(data))
	mode.CryptBlocks(encrypted, data)

	// Adding an initialization vector to encrypted data
	encrypted = append(iv, encrypted...)

	return encrypted, nil
}

func (e *Cipher) Decrypt(data []byte) error {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil
	}

	// Checking that the encrypted data is of sufficient length
	if len(data) < aes.BlockSize {
		return ErrDataIsNotLongEnough
	}

	// We extract the initialization vector from the encrypted data (at the beginning)
	iv := data[:aes.BlockSize]
	encrypted := data[aes.BlockSize:]

	// Create a CBC decryption mode using the AES block cipher
	mode := cipher.NewCBCDecrypter(block, iv)

	// Decrypting the data
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// Copying the decrypted data to the original buffer
	copy(data, decrypted)

	return nil
}
