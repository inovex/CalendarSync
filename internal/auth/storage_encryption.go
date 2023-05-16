package auth

import (
	"fmt"
	"io"

	"filippo.io/age"
)

// Ensure EncryptedFile implements the io.ReadWriteCloser interface
var _ io.ReadWriteCloser = &EncryptedFile{}

// EncryptedFile offers
type EncryptedFile struct {
	upstream   io.ReadWriter
	passphrase string
	decryptor  io.Reader
	encryptor  io.WriteCloser
}

// NewEncryptedFile takes the upstream io.ReadWriteCloser and a passphrase to setup an EncryptedFile
// The actual encryption or decryption setup will happen lazy on the first read or write
func NewEncryptedFile(file io.ReadWriter, passphrase string) *EncryptedFile {
	return &EncryptedFile{
		upstream:   file,
		passphrase: passphrase,
	}
}

// Read implement the io.Reader interface.
// On first call, this sets up the age decryption infrastructure
func (e *EncryptedFile) Read(p []byte) (n int, err error) {
	if e.decryptor == nil {
		identity, err := age.NewScryptIdentity(e.passphrase)
		if err != nil {
			return 0, fmt.Errorf("failed to create age identity: %w", err)
		}
		d, err := age.Decrypt(e.upstream, identity)
		if err != nil {
			return 0, fmt.Errorf("failed to setup data decryption: %w", err)
		}
		e.decryptor = d
	}
	return e.decryptor.Read(p)
}

// Write implement the io.Writer interface.
// On first call, this sets up the age encryption infrastructure
func (e *EncryptedFile) Write(p []byte) (n int, err error) {
	if e.encryptor == nil {
		recipent, err := age.NewScryptRecipient(e.passphrase)
		if err != nil {
			return 0, fmt.Errorf("failed to create age recipient: %w", err)
		}
		encryptor, err := age.Encrypt(e.upstream, recipent)
		if err != nil {
			return 0, fmt.Errorf("failed to setup encryption: %w", err)
		}
		e.encryptor = encryptor
	}
	return e.encryptor.Write(p)
}

// Close implements the io.Closer interface
// If ever data was encrypted, this closes the encryption stream
func (e *EncryptedFile) Close() error {
	if e.encryptor != nil {
		return e.encryptor.Close()

	}
	return nil
}
