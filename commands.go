package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/eighty4/maestro/composable"
	"github.com/eighty4/maestro/util"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type CommandType string

const (
	CargoCommand     CommandType = "Rust"
	DockerCompose    CommandType = "DockerCompose"
	NpmScript        CommandType = "Npm"
	SpringBootGradle CommandType = "SpringBootGradle"
	SpringBootMaven  CommandType = "SpringBootMaven"
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

func findSpringBootGradle(dir string) []Command {
	buildGradlePath := filepath.Join(dir, "build.gradle")
	if !util.IsFile(buildGradlePath) {
		return nil
	}
	buildGradle, err := os.ReadFile(buildGradlePath)
	if err != nil {
		log.Fatalln(err)
	}
	if !strings.Contains(string(buildGradle), "org.springframework.boot") {
		return nil
	}
	gradleBin := "gradle"
	if runtime.GOOS == "windows" {
		gradlewBatBin := filepath.Join(dir, "gradlew.bat")
		if util.IsFile(gradlewBatBin) {
			gradleBin = gradlewBatBin
		}
	} else {
		gradlewBin := filepath.Join(dir, "gradlew")
		if util.IsFile(gradlewBin) {
			gradleBin = gradlewBin
		}
	}
	return []Command{{
		name: "Gradle Spring Boot run",
		desc: "Gradle Spring Boot run",
		process: func() *composable.Process {
			return composable.NewProcess(gradleBin, []string{"bootRun"}, dir)
		},
	}}
}

type (
	mavenProject struct {
		Build *mavenBuild `xml:"build"`
	}

	mavenBuild struct {
		Plugins *mavenPlugins `xml:"plugins"`
	}

	mavenPlugins struct {
		Plugins []*mavenPlugin `xml:"plugin"`
	}

	mavenPlugin struct {
		GroupId    string `xml:"groupId"`
		ArtifactId string `xml:"artifactId"`
	}
)

func findSpringBootMaven(dir string) []Command {
	pomXmlPath := filepath.Join(dir, "pom.xml")
	if !util.IsFile(pomXmlPath) {
		return nil
	}
	pomXml, err := os.ReadFile(pomXmlPath)
	if err != nil {
		log.Fatalln(err)
	}
	var pom mavenProject
	if err = xml.Unmarshal(pomXml, &pom); err != nil {
		log.Fatalln(err)
	}
	springBoot := false
	for _, plugin := range pom.Build.Plugins.Plugins {
		if plugin.GroupId == "org.springframework.boot" && plugin.ArtifactId == "spring-boot-maven-plugin" {
			springBoot = true
		}
	}
	if !springBoot {
		return nil
	}
	mavenBin := "mvn"
	if runtime.GOOS == "windows" {
		mvnwCmdBin := filepath.Join(dir, "mvnw.cmd")
		if util.IsFile(mvnwCmdBin) {
			mavenBin = mvnwCmdBin
		}
	} else {
		mvnwBin := filepath.Join(dir, "gradlew")
		if util.IsFile(mvnwBin) {
			mavenBin = mvnwBin
		}
	}
	return []Command{{
		name: "Maven Spring Boot run",
		desc: "Maven Spring Boot run",
		process: func() *composable.Process {
			return composable.NewProcess(mavenBin, []string{"spring-boot:run"}, dir)
		},
	}}
}

func lsCommands() {
	cwd := util.Cwd()
	cmds := make(map[CommandType][]Command)
	cmds[CargoCommand] = findCargoCommands(cwd)
	cmds[DockerCompose] = findDockerCompose(cwd)
	cmds[NpmScript] = findNpmScripts(cwd)
	cmds[SpringBootGradle] = findSpringBootGradle(cwd)
	cmds[SpringBootMaven] = findSpringBootMaven(cwd)
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
	if len(pkg.commands[SpringBootGradle]) > 0 {
		fmt.Printf("/%s/build.gradle\n", pkg.name)
		for range pkg.commands[SpringBootGradle] {
			fmt.Println(" gradlew bootRun")
		}
	}
	if len(pkg.commands[SpringBootMaven]) > 0 {
		fmt.Printf("/%s/pom.xml\n", pkg.name)
		for range pkg.commands[SpringBootMaven] {
			fmt.Println(" mvnw spring-boot:run")
		}
	}
}
