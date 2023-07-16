package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
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
	commands []*Command
	dir      string
	name     string
}

type Command struct {
	Archetype string
	Desc      string
	Dir       string
	Exec      *composable.ExecDescription
	File      string
	Id        string
	Name      string
}

type CommandOptions struct {
	Desc string
	Dir  string
	Exec string
	Id   string
	Name string
}

func NewCommand(cmdOpts *CommandOptions) (*Command, error) {
	if len(cmdOpts.Dir) == 0 {
		return nil, errors.New("must specify a command dir")
	}

	if len(cmdOpts.Exec) > 0 && len(cmdOpts.Id) > 0 {
		return nil, errors.New("should only specify an exec string or an id but not both")
	}

	if len(cmdOpts.Exec) > 0 {
		// todo check if exec string matches a CommandArchetype with a fn of (composable.ExecDescription) => bool
		exec := composable.ParseCmdString(cmdOpts.Exec, cmdOpts.Dir)
		name := cmdOpts.Name
		if len(name) == 0 {
			name = exec.Binary
		}
		return &Command{
			Desc: cmdOpts.Desc,
			Dir:  cmdOpts.Dir,
			Exec: exec,
			Name: name,
		}, nil
	}

	if len(cmdOpts.Id) > 0 {
		cmd, err := ParseCommandId(cmdOpts.Id, cmdOpts.Dir)
		if err != nil {
			return nil, errors.New("error parsing command id: " + err.Error())
		}
		if len(cmd.Desc) == 0 {
			cmd.Desc = cmdOpts.Desc
		}
		if len(cmdOpts.Name) > 0 {
			cmd.Name = cmdOpts.Name
		}
		return cmd, nil
	}

	return nil, errors.New("does not specify an exec string or an id")
}

func ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

// todo extensible configuration driven CommandArchetype impl
// todo go.mod => go run
// todo gradle application plugin
// todo procfile
// todo pubspec.yaml => dart run
// todo CommandArchetype registry service and initialization
type CommandArchetype interface {
	ArchetypeId() string
	FindCommands(dir string) []*Command
	ParseCommandId(id string, dir string) (*Command, error)
}

type CargoCommandArchetype struct {
}

func (a *CargoCommandArchetype) ArchetypeId() string {
	return "cargo:run"
}

func (a *CargoCommandArchetype) FindCommands(dir string) []*Command {
	cargoTomlPath := filepath.Join(dir, "Cargo.toml")
	if !util.IsFile(cargoTomlPath) {
		return nil
	}
	// todo parse Cargo.toml and create commands for each binary target
	return []*Command{{
		Archetype: a.ArchetypeId(),
		Dir:       dir,
		Exec: &composable.ExecDescription{
			Binary: "cargo",
			Args:   []string{"run"},
			Dir:    dir,
		},
		File: "Cargo.toml",
		Id:   a.ArchetypeId(),
		Name: "run",
	}}
}

func (a *CargoCommandArchetype) ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

type DockerComposeArchetype struct {
}

func (a *DockerComposeArchetype) ArchetypeId() string {
	return "docker:compose"
}

func (a *DockerComposeArchetype) FindCommands(dir string) []*Command {
	customFileNameRegex, err := regexp.Compile(`^(?:.+)?docker-compose(?:.+)?\.ya?ml$`)
	if err != nil {
		log.Fatalln(err)
	}
	defaultFileNameRegex, err := regexp.Compile(`^docker-compose\.ya?ml$`)
	if err != nil {
		log.Fatalln(err)
	}
	var dockerComposeFiles []string
	if files, err := os.ReadDir(dir); err != nil {
		log.Fatalln(err)
	} else {
		for _, file := range files {
			if !file.IsDir() && customFileNameRegex.MatchString(file.Name()) {
				dockerComposeFiles = append(dockerComposeFiles, file.Name())
			}
		}
	}
	var cmds []*Command
	for _, dockerComposeFile := range dockerComposeFiles {
		dockerComposeFile := dockerComposeFile
		args := []string{"compose", "up", "-d"}
		if !defaultFileNameRegex.MatchString(dockerComposeFile) {
			args = append(args, "-f", dockerComposeFile)
		}
		cmds = append(cmds, &Command{
			Archetype: a.ArchetypeId(),
			Dir:       dir,
			Exec: &composable.ExecDescription{
				Binary: "docker",
				Args:   args,
				Dir:    dir,
			},
			File: dockerComposeFile,
			Id:   a.ArchetypeId() + ":" + dockerComposeFile,
			Name: dockerComposeFile,
		})
	}
	return cmds
}

func (a *DockerComposeArchetype) ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

type NpmScriptArchetype struct {
}

func (a *NpmScriptArchetype) ArchetypeId() string {
	return "npm:run"
}

func (a *NpmScriptArchetype) FindCommands(dir string) []*Command {
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
	var cmds []*Command
	for scriptName := range scripts {
		if len(scriptName) > 3 && scriptName[:3] == "pre" {
			continue
		}
		cmds = append(cmds, &Command{
			Archetype: a.ArchetypeId(),
			Dir:       dir,
			Exec: &composable.ExecDescription{
				// todo resolve pnpm, yarn?
				Binary: "npm",
				Args:   []string{"run", scriptName},
				Dir:    dir,
			},
			File: "package.json",
			Id:   a.ArchetypeId() + ":" + scriptName,
			Name: scriptName,
		})
	}
	return cmds
}

func (a *NpmScriptArchetype) ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

type GradleSpringBootArchetype struct {
}

func (a *GradleSpringBootArchetype) ArchetypeId() string {
	return "gradle:spring"
}

func (a *GradleSpringBootArchetype) FindCommands(dir string) []*Command {
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
	return []*Command{{
		Archetype: a.ArchetypeId(),
		Dir:       dir,
		Exec: &composable.ExecDescription{
			Binary: gradleBin,
			Args:   []string{"bootRun"},
			Dir:    dir,
		},
		File: "build.gradle",
		Id:   a.ArchetypeId() + ":bootRun",
		Name: "bootRun",
	}}
}

func (a *GradleSpringBootArchetype) ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

type (
	MavenSpringBootArchetype struct {
	}

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

func (a *MavenSpringBootArchetype) ArchetypeId() string {
	return "maven:spring"
}

func (a *MavenSpringBootArchetype) FindCommands(dir string) []*Command {
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
	return []*Command{{
		Archetype: a.ArchetypeId(),
		Dir:       dir,
		Exec: &composable.ExecDescription{
			Binary: mavenBin,
			Args:   []string{"spring-boot:run"},
			Dir:    dir,
		},
		File: "pom.xml",
		Id:   a.ArchetypeId() + ":bootRun",
		Name: "bootRun",
	}}
}

func (a *MavenSpringBootArchetype) ParseCommandId(id string, dir string) (*Command, error) {
	// todo implement me
	panic("implement me")
}

func ScanForPackages(rootDir string, packageScanDepth int) ([]*Package, error) {
	log.Printf("[TRACE] ScanForPackages(\"%s\", %d)\n", rootDir, packageScanDepth)
	dirs := append(util.Subdirectories(rootDir, packageScanDepth), rootDir)
	done := make(chan error)
	c := make(chan *Package)
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
			var cmds []*Command
			archetypes := []CommandArchetype{
				&CargoCommandArchetype{},
				&DockerComposeArchetype{},
				&GradleSpringBootArchetype{},
				&MavenSpringBootArchetype{},
				&NpmScriptArchetype{},
			}
			for _, archetype := range archetypes {
				cmds = append(cmds, archetype.FindCommands(dir)...)
			}
			sort.Slice(cmds, func(i, j int) bool {
				return cmds[i].Id < cmds[j].Id
			})
			if len(cmds) > 0 {
				c <- &Package{
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

	var result []*Package
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

func printCommands(packages []*Package) {
	for _, pkg := range packages {
		pad := ""
		if pkg.name != "" {
			pad = " "
			fmt.Println(pkg.name)
		}
		for _, cmd := range pkg.commands {
			fmt.Printf("%s%s\n", pad, cmd.Id)
		}
	}
}
