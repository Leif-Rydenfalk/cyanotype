package digest

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/tychonis/cyanotype/internal/serializer"
	"github.com/tychonis/cyanotype/model"
)

func SHA256FromReader(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", err
	}

	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum), nil
}

func SHA256FromSymbol(s model.ConcreteSymbol) (string, error) {
	data, err := serializer.Serialize(s)
	if err != nil {
		return "", err
	}
	return SHA256FromReader(bytes.NewReader(data))
}

func SHA256FromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return SHA256FromReader(f)
}

func RandomSHA256() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	sum := sha256.Sum256(randomBytes)
	return hex.EncodeToString(sum[:]), nil
}
