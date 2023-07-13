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

type Package struct {
	commands []Command
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
			desc: "cargo " + cmd,
			name: "cargo:" + cmd,
			process: func() *composable.Process {
				return composable.NewProcess("cargo", []string{cmd}, dir)
			},
		})
	}
	return cmds
}

func findDockerCompose(dir string) []Command {
	customFileNameRegex, err := regexp.Compile(`^(?:.+)?docker-compose(?:.+)?\.ya?ml$`)
	if err != nil {
		log.Fatalln(err)
	}
	defaultFileNameRegex, err := regexp.Compile(`^docker-compose\.ya?ml$`)
	if err != nil {
		log.Fatalln(err)
	}
	files, err := os.ReadDir(dir)
	var dockerComposeFiles []string
	if err != nil {
		log.Fatalln(err)
	} else {
		for _, file := range files {
			if !file.IsDir() && customFileNameRegex.MatchString(file.Name()) {
				dockerComposeFiles = append(dockerComposeFiles, file.Name())
			}
		}
	}
	var cmds []Command
	for _, dockerComposeFile := range dockerComposeFiles {
		dockerComposeFile := dockerComposeFile
		desc := "docker compose up"
		if !defaultFileNameRegex.MatchString(dockerComposeFile) {
			desc = desc + " -f " + dockerComposeFile
		}
		cmds = append(cmds, Command{
			desc: desc,
			name: "docker:compose:" + dockerComposeFile,
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
	for scriptName := range scripts {
		if len(scriptName) > 3 && scriptName[:3] == "pre" {
			continue
		}
		cmds = append(cmds, Command{
			desc: "npm run " + scriptName,
			name: "npm:run:" + scriptName,
			process: func() *composable.Process {
				// todo resolve pnpm, yarn?
				return composable.NewProcess("npm", []string{"run", scriptName}, dir)
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
		desc: filepath.Base(gradleBin) + " bootRun",
		name: "gradle:spring-boot:run",
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
		mvnwBin := filepath.Join(dir, "mvnw")
		if util.IsFile(mvnwBin) {
			mavenBin = mvnwBin
		}
	}
	return []Command{{
		name: "maven:spring-boot:run",
		desc: filepath.Base(mavenBin) + " spring-boot:run",
		process: func() *composable.Process {
			return composable.NewProcess(mavenBin, []string{"spring-boot:run"}, dir)
		},
	}}
}

func ScanForPackages(rootDir string, packageScanDepth int) ([]Package, error) {
	log.Printf("[TRACE] ScanForPackages(\"%s\", %d)\n", rootDir, packageScanDepth)
	dirs := append(util.Subdirectories(rootDir, packageScanDepth), rootDir)
	done := make(chan error)
	c := make(chan Package)
	wg := sync.WaitGroup{}
	wg.Add(len(dirs))

	for _, dir := range dirs {
		dir := dir
		go func() {
			name := ""
			if len(rootDir) != len(dir) {
				if rel, err := filepath.Rel(rootDir, dir); err != nil {
					done <- err
				} else {
					name += rel
				}
			}
			var cmds []Command
			cmds = append(cmds, findCargoCommands(dir)...)
			cmds = append(cmds, findDockerCompose(dir)...)
			cmds = append(cmds, findNpmScripts(dir)...)
			cmds = append(cmds, findSpringBootGradle(dir)...)
			cmds = append(cmds, findSpringBootMaven(dir)...)
			sort.Slice(cmds, func(i, j int) bool {
				return cmds[i].name < cmds[j].name
			})
			if len(cmds) > 0 {
				c <- Package{
					commands: cmds,
					dir:      dir,
					name:     name,
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
		case err := <-done:
			close(c)
			close(done)
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] ScanForPackages(\"%s\", %d) found %d %s\n", rootDir, packageScanDepth, len(result), util.PluralPrint("package", len(result)))
			sort.Slice(result, func(i, j int) bool {
				return result[i].name == "" || result[i].name < result[j].name
			})
			return result, nil
		}
	}
}

func lsCommands() {
	packages, err := ScanForPackages(util.Cwd(), 2)
	if err != nil {
		log.Fatalln(err.Error())
	}
	printCommands(packages)
}

func printCommands(packages []Package) {
	for _, pkg := range packages {
		pad := ""
		if pkg.name != "" {
			pad = " "
			fmt.Println(pkg.name)
		}
		for _, cmd := range pkg.commands {
			fmt.Printf("%s%s\n", pad, cmd.desc)
		}
	}
}
