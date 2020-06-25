package box

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const (
	releasePrefix = "https://github.com/KiloProjects/isolate/releases/download/v2.0/"
	configURL     = releasePrefix + "default.cf"
	configPath    = "/usr/local/etc/isolate"
	isolateURL    = releasePrefix + "isolate"
	isolatePath   = "/data/isolate"
)

// Directory represents a directory rule
type Directory struct {
	In      string
	Out     string
	Opts    string
	Removes bool
}

// Env represents a variable-value pair for an environment variable
type Env struct {
	Var   string
	Value string
}

// Config is the struct that controls the sandbox
type Config struct {
	// Box ID
	ID int

	// Mark if Cgroups should be enabled
	Cgroups bool
	// Maximum Cgroup memory (in kbytes)
	CgroupMem int

	// Directories represents the list of mounted directories
	Directories []Directory

	// Environment
	InheritEnv   bool
	EnvToInherit []string
	EnvToSet     map[string]string

	// Time limits (in seconds)
	TimeLimit      float64
	WallTimeLimit  float64
	ExtraTimeLimit float64

	InputFile  string
	OutputFile string
	ErrFile    string
	MetaFile   string

	// Memory limits (in kbytes)
	MemoryLimit int
	StackSize   int

	// Processes represents the maximum number of processes the program can create
	Processes int

	// Chdir is the directory (relative to the box root) to switch in
	Chdir string
}

// Box is the struct for the current box
type Box struct {
	path   string
	Config Config

	// Debug prints additional info
	Debug bool
}

// BuildRunFlags compiles all flags into an array
func (c *Config) BuildRunFlags() (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(c.ID))

	if c.Cgroups {
		res = append(res, "--cg", "--cg-timing")
		if c.CgroupMem != 0 {
			res = append(res, "--cg-mem="+strconv.Itoa(c.CgroupMem))
		}
	}

	for _, dir := range c.Directories {
		if dir.Removes {
			res = append(res, "--dir="+dir.In+"=")
			continue
		}
		toAdd := "--dir="
		toAdd += dir.In + "="
		if dir.Out == "" {
			toAdd += dir.In
		} else {
			toAdd += dir.Out
		}
		if dir.Opts != "" {
			toAdd += ":" + dir.Opts
		}
		res = append(res, toAdd)
	}

	if c.InheritEnv {
		res = append(res, "--full-env")
	}
	for _, env := range c.EnvToInherit {
		res = append(res, "--env="+env)
	}
	for key, val := range c.EnvToSet {
		res = append(res, "--env="+key+"="+val)
	}

	if c.TimeLimit != 0 {
		res = append(res, "--time="+strconv.FormatFloat(c.TimeLimit, 'f', -1, 64))
	}
	if c.WallTimeLimit != 0 {
		res = append(res, "--wall-time="+strconv.FormatFloat(c.WallTimeLimit, 'f', -1, 64))
	}
	if c.ExtraTimeLimit != 0 {
		res = append(res, "--extra-time="+strconv.FormatFloat(c.ExtraTimeLimit, 'f', -1, 64))
	}

	if c.MemoryLimit != 0 {
		res = append(res, "--mem="+strconv.Itoa(c.MemoryLimit))
	}
	if c.StackSize != 0 {
		res = append(res, "--stack="+strconv.Itoa(c.StackSize))
	}

	if c.Processes == 0 {
		res = append(res, "--processes")
	} else {
		res = append(res, "--processes="+strconv.Itoa(c.Processes))
	}

	if c.InputFile != "" {
		res = append(res, "--stdin="+c.InputFile)
	}
	if c.OutputFile != "" {
		res = append(res, "--stdout="+c.OutputFile)
	}
	if c.ErrFile != "" {
		res = append(res, "--stderr="+c.ErrFile)
	}
	if c.MetaFile != "" {
		res = append(res, "--meta="+c.MetaFile)
	}

	if c.Chdir != "" {
		res = append(res, "--chdir="+c.Chdir)
	}
	res = append(res, "--silent", "--run", "--")
	return
}

// WriteFile writes a file to the specified filepath inside the box
func (b *Box) WriteFile(filepath, data string) error {
	return ioutil.WriteFile(path.Join(b.path, filepath), []byte(data), 0777)
}

// RemoveFile tries to remove a created file from inside the sandbox
func (b *Box) RemoveFile(filepath string) error {
	return os.Remove(path.Join(b.path, filepath))
}

// GetFile returns a file from inside the sandbox
func (b *Box) GetFile(filepath string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(b.path, filepath))
}

// Cleanup is a convenience wrapper for cleanupBox
func (b *Box) Cleanup() error {
	var params []string
	if b.Config.Cgroups {
		params = append(params, "--cg")
	}
	params = append(params, "--box-id="+strconv.Itoa(b.Config.ID), "--cleanup")
	return exec.Command(isolatePath, params...).Run()
}

// ExecCommand runs a command
func (b *Box) ExecCommand(command ...string) (string, string, error) {
	return b.ExecWithStdin("", command...)
}

// ExecCombinedOutput runs a command and returns the combined output
func (b *Box) ExecCombinedOutput(command ...string) ([]byte, error) {
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command(isolatePath, params...)
	if b.Debug {
		fmt.Println("DEBUG:", cmd.String())
	}
	return cmd.CombinedOutput()
}

// ExecWithStdin runs a command with a specified stdin
// Returns the stdout, stderr and error (if anything bad happened)
func (b *Box) ExecWithStdin(stdin string, command ...string) (string, string, error) {
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command(isolatePath, params...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	if b.Debug {
		fmt.Println("DEBUG:", cmd.String())
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// NewBox returns a new box instance from the specified ID
func NewBox(config Config) (*Box, error) {
	var needsSetup bool

	if _, err := os.Stat(isolatePath); os.IsNotExist(err) {
		needsSetup = true
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		needsSetup = true
	}
	// if exist != 0b11, then we need to setup isolate
	if needsSetup {
		if err := setupIsolate(); err != nil {
			panic("Can't setup isolate: " + err.Error())
		}
	}

	ret, err := exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", config.ID), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", config.ID), "--cleanup").Run()
		return NewBox(config)
	}

	if err != nil {
		return nil, err
	}

	return &Box{path: strings.TrimSpace(string(ret)), Config: config}, nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	return nil
}

func setupIsolate() error {
	// download isolate
	if err := downloadFile(isolateURL, isolatePath); err != nil {
		return err
	}

	// download the config file
	if err := downloadFile(configURL, configPath); err != nil {
		return err
	}

	return nil
}
