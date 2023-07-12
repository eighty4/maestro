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
	"sort"
	"strings"
	"sync"
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
	if packageJsonMap["scripts"] == nil {
		return nil
	}
	scripts, ok := packageJsonMap["scripts"].(map[string]interface{})
	if !ok {
		return nil
	}
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

func ScanForPackages(rootDir string, packageScanDepth int) []Package {
	log.Printf("[TRACE] ScanForPackages(\"%s\", %d)\n", rootDir, packageScanDepth)
	dirs := append(util.Subdirectories(rootDir, packageScanDepth), rootDir)
	done := make(chan interface{})
	c := make(chan Package)
	wg := sync.WaitGroup{}
	wg.Add(len(dirs))

	for _, dir := range dirs {
		dir := dir
		go func() {
			name := ""
			if len(rootDir) != len(dir) {
				if rel, err := filepath.Rel(rootDir, dir); err != nil {
					log.Fatalln(err)
				} else {
					name += rel
				}
			}
			cmdMap := make(map[CommandType][]Command)
			cmdMap[CargoCommand] = findCargoCommands(dir)
			cmdMap[DockerCompose] = findDockerCompose(dir)
			cmdMap[NpmScript] = findNpmScripts(dir)
			cmdMap[SpringBootGradle] = findSpringBootGradle(dir)
			cmdMap[SpringBootMaven] = findSpringBootMaven(dir)
			for _, cmds := range cmdMap {
				if len(cmds) > 0 {
					c <- Package{
						commands: cmdMap,
						dir:      dir,
						name:     name,
					}
					break
				}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		done <- nil
	}()

	var result []Package
	for {
		select {
		case p := <-c:
			result = append(result, p)
			break
		case <-done:
			close(c)
			close(done)
			log.Printf("[DEBUG] ScanForPackages(\"%s\", %d) found %d %s\n", rootDir, packageScanDepth, len(result), util.PluralPrint("package", len(result)))
			sort.Slice(result, func(i, j int) bool {
				return result[i].name == "" || result[i].name < result[j].name
			})
			return result
		}
	}
}

func lsCommands() {
	packages := ScanForPackages(util.Cwd(), 2)
	for _, pkg := range packages {
		pad := ""
		if pkg.name != "" {
			pad = " "
			fmt.Println(pkg.name)
		}
		if len(pkg.commands[CargoCommand]) > 0 {
			for _, cargoCommand := range pkg.commands[CargoCommand] {
				fmt.Printf("%scargo %s\n", pad, cargoCommand.name)
			}
		}
		if len(pkg.commands[DockerCompose]) > 0 {
			for _, dockerCompose := range pkg.commands[DockerCompose] {
				if strings.Index(dockerCompose.name, "docker-compose.") == 0 {
					fmt.Printf("%sdocker compose up\n", pad)
				} else {
					fmt.Printf("%sdocker compose up -f %s\n", pad, dockerCompose.name)
				}
			}
		}
		if len(pkg.commands[NpmScript]) > 0 {
			for _, npmScript := range pkg.commands[NpmScript] {
				fmt.Printf("%snpm run %s\n", pad, npmScript.name)
			}
		}
		if len(pkg.commands[SpringBootGradle]) > 0 {
			for range pkg.commands[SpringBootGradle] {
				fmt.Printf("%sgradlew bootRun\n", pad)
			}
		}
		if len(pkg.commands[SpringBootMaven]) > 0 {
			for range pkg.commands[SpringBootMaven] {
				fmt.Printf("%smvnw spring-boot:run\n", pad)
			}
		}
	}
}
