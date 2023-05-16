package auth

import (
	"bytes"
	"io"
	"testing"
)

type testFile struct {
	bytes.Buffer
}

func (t *testFile) Close() error {
	return nil
}

func TestEncryption(t *testing.T) {
	text := []byte("Lorem ipsum dolor sit amet")
	passphrase := "I like calendarsync"

	file := &testFile{}
	eFile := NewEncryptedFile(file, passphrase)
	written, err := eFile.Write(text)
	if err != nil {
		t.Fatalf("got error '%v', expected nil", err)
	}
	if len(text) != written {
		t.Fatalf("expected %d bytes to be written, got %d", len(text), written)
	}
	err = eFile.Close()
	if err != nil {
		t.Fatalf("got error '%v', expected nil", err)
	}

	eFile = NewEncryptedFile(file, passphrase)
	data, err := io.ReadAll(eFile)
	if err != nil {
		t.Fatalf("got error '%v', expected nil", err)
	}
	if string(data) != string(text) {
		t.Fatalf("got: %s, expected %s", string(data), string(text))
	}
}
