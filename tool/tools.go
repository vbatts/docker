package tool

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dotcloud/docker/pkg/system"
	"github.com/dotcloud/docker/utils"
)

var (
	DefaultDir = Dir("")
	toolPrefix = "docker-"
)

type ToolDir string

func (td ToolDir) Exec(cmd string, args []string) error {
	return system.Exec(filepath.Join(string(td), cmd), args, os.Environ())
}

func (td ToolDir) Tools() ([]string, error) {
	cmds := []string{}
	matches, err := filepath.Glob(filepath.Join(string(td), toolPrefix+"*"))
	if err != nil {
		return cmds, err
	}
	for _, m := range matches {
		cmds = append(cmds, strings.TrimPrefix(filepath.Base(m), toolPrefix))
	}
	return cmds, nil
}

func Exec(cmd string, args []string) error {
	return DefaultDir.Exec(cmd, args)
}

func Dir(localcopy string) ToolDir {
	return ToolDir(filepath.Join(filepath.Dir(utils.DockerInitPath(localcopy)), "tool"))
}

func Tools() ([]string, error) {
	return DefaultDir.Tools()
}
