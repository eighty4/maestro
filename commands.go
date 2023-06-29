package main

import (
	"encoding/json"
	"fmt"
	"github.com/eighty4/maestro/composable"
	"github.com/eighty4/maestro/util"
	"log"
	"os"
	"path/filepath"
)

type CommandType string

const (
	NpmScript CommandType = "NpmScript"
)

type Package struct {
	commands map[CommandType][]Command
	dir      string
	name     string
}

type Command struct {
	desc    string
	name    string
	process func() *composable.Process
}

func findNpmScripts(dir string) []Command {
	packageJsonPath := filepath.Join(dir, "package.json")
	if !util.IsFile(packageJsonPath) {
		return nil
	}
	packageJsonString, err := os.ReadFile(packageJsonPath)
	if err != nil {
		log.Fatalln(err)
	}
	var packageJsonMap map[string]interface{}
	if err = json.Unmarshal(packageJsonString, &packageJsonMap); err != nil {
		log.Fatalln(err)
	}
	scripts := packageJsonMap["scripts"].(map[string]interface{})
	if len(scripts) < 1 {
		return nil
	}
	var cmds []Command
	for script := range scripts {
		cmds = append(cmds, Command{
			desc: scripts[script].(string),
			name: script,
			process: func() *composable.Process {
				// todo resolve pnpm, yarn?
				return composable.NewProcess("npm", []string{"run", script}, dir)
			},
		})
	}
	return cmds
}

func lsCommands() {
	cwd := util.Cwd()
	cmds := make(map[CommandType][]Command)
	cmds[NpmScript] = findNpmScripts(cwd)
	pkg := Package{
		commands: cmds,
		dir:      cwd,
		name:     cwd,
	}
	if len(pkg.commands[NpmScript]) > 0 {
		for _, npmScript := range pkg.commands[NpmScript] {
			fmt.Printf("%s (%s)\n", npmScript.name, npmScript.desc)
		}
	}
}
