package datamanager

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/AlexVasiluta/kilonova/models"
)

// Manager helps open the files in the data directory, this is supposed to be data that should not be stored in the database because it's a binary blob
type Manager struct {
	RootPath string
}

// Session holds the session data of a specified user
type Session struct {
	User    models.User
	Expires time.Time
}

// GetTest returns a test buffer for the specified problem
func (m *Manager) GetTest(pbID int, testID int) (string, string, error) {
	input, err := ioutil.ReadFile(path.Join(m.RootPath, strconv.Itoa(pbID), "input", strconv.Itoa(testID)+".txt"))
	if err != nil {
		return "", "", err
	}
	output, err := ioutil.ReadFile(path.Join(m.RootPath, strconv.Itoa(pbID), "output", strconv.Itoa(testID)+".txt"))
	if err != nil {
		return "", "", err
	}
	return string(input), string(output), err
}

// SaveTest saves an (input, output) pair of strings to disk to be used later as tests
func (m *Manager) SaveTest(pbID int, testID int, input []byte, output []byte) error {
	if err := os.MkdirAll(path.Join(m.RootPath, strconv.Itoa(pbID), "input"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(m.RootPath, strconv.Itoa(pbID), "output"), 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(m.RootPath, strconv.Itoa(pbID), "input", strconv.Itoa(testID)+".txt"),
		input, 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(m.RootPath, strconv.Itoa(pbID), "output", strconv.Itoa(testID)+".txt"),
		output, 0777); err != nil {
		return err
	}
	return nil
}

// GetAttachment returns an attachment from disk
func (m *Manager) GetAttachment(dir string, ID int, name string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(m.RootPath, dir, strconv.Itoa(ID), "attachment", name))
}

// SaveAttachment saves an attachment (ex: image) of something (specified with dir, right now the only thing you are should do is "problem") to disk
func (m *Manager) SaveAttachment(dir string, ID int, data []byte, name string) error {
	return ioutil.WriteFile(path.Join(m.RootPath, dir, strconv.Itoa(ID), "attachment", name), data, 0777)
}

// NewManager returns a new manager instance
func NewManager(path string) *Manager {
	os.MkdirAll(path, 0777)
	return &Manager{RootPath: path}
}

// GetSession returns a session based on an ID
func (m *Manager) GetSession(id string) Session {
	return Session{}
}

// AddSession adds a session
func (m *Manager) AddSession(session Session) string {
	return ""
}
