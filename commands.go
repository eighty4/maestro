package main

import (
	"encoding/json"
	"fmt"
	"github.com/eighty4/maestro/composable"
	"github.com/eighty4/maestro/util"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type CommandType string

const (
	CargoCommand  CommandType = "Rust"
	DockerCompose CommandType = "DockerCompose"
	NpmScript     CommandType = "Npm"
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

func findCargoCommands(dir string) []Command {
	cargoTomlPath := filepath.Join(dir, "Cargo.toml")
	if !util.IsFile(cargoTomlPath) {
		return nil
	}
	var cmds []Command
	for _, cmd := range []string{"test", "run"} {
		cmd := cmd
		cmds = append(cmds, Command{
			desc: cmd,
			name: cmd,
			process: func() *composable.Process {
				return composable.NewProcess("echo", []string{cmd}, dir)
			},
		})
	}
	return cmds
}

func findDockerCompose(dir string) []Command {
	regex, err := regexp.Compile(`^(?:.+)?docker-compose(?:.+)?\.ya?ml$`)
	if err != nil {
		log.Fatalln(err)
	}
	files, err := os.ReadDir(dir)
	var dockerComposeFiles []string
	if err != nil {
		log.Fatalln(err)
	} else {
		for _, file := range files {
			if !file.IsDir() && regex.Match([]byte(file.Name())) {
				dockerComposeFiles = append(dockerComposeFiles, file.Name())
			}
		}
	}
	var cmds []Command
	for _, dockerComposeFile := range dockerComposeFiles {
		dockerComposeFile := dockerComposeFile
		cmds = append(cmds, Command{
			desc: "docker compose up -d -f " + dockerComposeFile,
			name: dockerComposeFile,
			process: func() *composable.Process {
				return composable.NewProcess("docker", []string{"compose", "up", "-d", "-f", dockerComposeFile}, dir)
			},
		})
	}
	return cmds
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
		if len(script) > 3 && script[:3] == "pre" {
			continue
		}
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
	cmds[CargoCommand] = findCargoCommands(cwd)
	cmds[DockerCompose] = findDockerCompose(cwd)
	cmds[NpmScript] = findNpmScripts(cwd)
	pkg := Package{
		commands: cmds,
		dir:      cwd,
		name:     filepath.Base(cwd),
	}
	if len(pkg.commands[CargoCommand]) > 0 {
		fmt.Printf("/%s/Cargo.toml\n", pkg.name)
		for _, cargoCommand := range pkg.commands[CargoCommand] {
			fmt.Printf(" %s\n", cargoCommand.name)
		}
	}
	if len(pkg.commands[DockerCompose]) > 0 {
		for _, dockerCompose := range pkg.commands[DockerCompose] {
			fmt.Printf("/%s/%s\n", pkg.name, dockerCompose.name)
			fmt.Println(" up")
		}
	}
	if len(pkg.commands[NpmScript]) > 0 {
		fmt.Printf("/%s/package.json\n", pkg.name)
		for _, npmScript := range pkg.commands[NpmScript] {
			fmt.Printf(" %s\n", npmScript.name)
		}
	}
}
