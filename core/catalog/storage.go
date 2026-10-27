package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tychonis/cyanotype/internal/serializer"
	"github.com/tychonis/cyanotype/model"
)

type Storage interface {
	Save(digest model.Digest, data []byte) error
	Load(digest model.Digest) ([]byte, error)
	Exists(digest model.Digest) (bool, error)

	SaveArtifact(artifact *model.Artifact) error
}

// TODO: Consider replacing file-based local storage with SQLite.
// Extract the PostgreSQL catalog implementation from BomHub into a
// reusable SQL backend supporting both SQLite and PostgreSQL.
// BomHub should access the SQL catalog directly, while remote CLI
// clients continue using the HTTP API.
type LocalStorage struct{}

func digestToPath(digest string) (string, error) {
	if len(digest) < 2 {
		return "", errors.New("incorrect digest")
	}
	folder := digest[:2]
	return filepath.Join(".bpc", "objects", folder, digest), nil
}

func (ls *LocalStorage) Save(digest model.Digest, data []byte) error {
	path, err := digestToPath(digest)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return atomicWrite(path, data, 0o644)
}

func (ls *LocalStorage) Load(digest model.Digest) ([]byte, error) {
	path, err := digestToPath(digest)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (ls *LocalStorage) Exists(digest model.Digest) (bool, error) {
	path, err := digestToPath(digest)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func (ls *LocalStorage) SaveArtifact(artifact *model.Artifact) error {
	data, err := serializer.Serialize(artifact)
	if err != nil {
		return err
	}
	// Only save metadata for local store.
	return ls.Save(artifact.Digest, data)
}

type MemoryStore struct {
	storage map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		storage: make(map[string][]byte),
	}
}

func (m *MemoryStore) Save(digest model.Digest, data []byte) error {
	m.storage[digest] = data
	return nil
}

func (m *MemoryStore) Load(digest model.Digest) ([]byte, error) {
	data, ok := m.storage[digest]
	if !ok {
		return nil, errors.New("digest not found")
	}
	return data, nil
}

func (m *MemoryStore) Exists(digest model.Digest) (bool, error) {
	_, ok := m.storage[digest]
	return ok, nil
}

func (m *MemoryStore) SaveArtifact(artifact *model.Artifact) error {
	data, err := serializer.Serialize(artifact)
	if err != nil {
		return err
	}
	return m.Save(artifact.Digest, data)
}

type APIStore struct {
	endpoint string

	client *http.Client
}

func NewAPIStore(endpoint string, client *http.Client) *APIStore {
	return &APIStore{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client:   client,
	}
}

func (a *APIStore) Save(digest model.Digest, data []byte) error {
	url := fmt.Sprintf("%s/definition/%s", a.endpoint, digest)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return errors.New("error response saving symbol: " + resp.Status)
	}
	return nil
}

func (a *APIStore) Load(digest model.Digest) ([]byte, error) {
	url := fmt.Sprintf("%s/definition/%s", a.endpoint, digest)
	resp, err := a.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("error response loading symbol: " + resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (a *APIStore) SaveArtifact(artifact *model.Artifact) error {
	parsed, err := url.Parse(artifact.Source)
	if err != nil {
		return err
	}
	exists, err := a.Exists(artifact.Digest)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	switch parsed.Scheme {
	case "file":
		filepath := parsed.Host + parsed.Path
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer f.Close()
		return a.uploadObject(artifact.Digest, artifact.Filename, f)
	case "digest":
		return fmt.Errorf("non exist digest in remote")
	case "s3":
		return fmt.Errorf("s3 scheme not supported yet")
	default:
		return fmt.Errorf("scheme not supported: %s", parsed.Scheme)
	}
}

func (a *APIStore) uploadObject(key, name string, r io.Reader) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, r); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/object/%s", a.endpoint, key),
		&body,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s: %s", resp.Status, data)
	}

	return nil
}

func (a *APIStore) Exists(key string) (bool, error) {
	req, err := http.NewRequest(
		http.MethodHead,
		fmt.Sprintf("%s/object/%s", a.endpoint, key),
		nil,
	)
	if err != nil {
		return false, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("HEAD failed: %s", resp.Status)
	}
}
