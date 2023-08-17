package main

import (
	"errors"
	"fmt"
	"github.com/eighty4/maestro/util"
	"log"
	"math"
	"path/filepath"
)

var composeMenuKeyCmds []KeyCmd

func composeProject(cfg *Config) error {
	// todo support merging an existing maestro config's commands into compose job
	if cfg.FileExists {
		return fmt.Errorf("this directory already has a %s file", cfg.Filename)
	}
	j, err := NewComposeProjectJob(cfg)
	if err != nil {
		return err
	}
	composeMenuKeyCmds = []KeyCmd{Up, Down, Space, Enter}
	if err := j.start(); err != nil {
		return err
	}
	return nil
}

type ComposeProjectJob struct {
	cfg       *Config
	packages  []*Package
	uiLines   int
	curPkgI   int
	cursorI   int
	cmdCursor bool
	selected  [][]bool
	doneC     chan error
}

func NewComposeProjectJob(cfg *Config) (*ComposeProjectJob, error) {
	if packages, err := ScanForPackages(cfg.Dir, 2); err != nil {
		return nil, err
	} else {
		selected := make([][]bool, len(packages))
		for i, p := range packages {
			selected[i] = make([]bool, len(p.commands))
			for _, c := range p.commands {
				c.Desc = c.Exec.ToString()
			}
		}
		return &ComposeProjectJob{
			cfg:       cfg,
			packages:  packages,
			uiLines:   0,
			curPkgI:   0,
			cursorI:   0,
			cmdCursor: true,
			selected:  selected,
			doneC:     make(chan error),
		}, nil
	}
}

func (j *ComposeProjectJob) start() error {
	if len(j.packages) == 0 {
		fmt.Println("Maestro is composing a project and did not find any packages.")
		return errors.New("no packages found")
	}
	if len(j.packages) >= math.MaxUint8 {
		fmt.Printf("Maestro is composing a project and found %d %s -- that's too many!\n", len(j.packages), util.PluralPrint("package", len(j.packages)))
		return errors.New("too many packages found")
	}
	fmt.Printf("Maestro is composing a project and found %d packages.\n\n", len(j.packages))
	fmt.Print("┌ Select the commands you'd like Maestro to orchestrate:\n|\n")
	j.refreshInterface()

	err := <-j.doneC
	close(j.doneC)
	return err
}

func (j *ComposeProjectJob) refreshInterface() {
	util.ClearTermLines(j.uiLines)
	var lines []string
	for pkgI := 0; pkgI <= j.curPkgI; pkgI++ {
		pkg := j.packages[pkgI]
		diamond := "◇"
		if pkgI == j.curPkgI {
			diamond = "◆"
		}
		arrow := " "
		if !j.cmdCursor && j.cursorI == pkgI {
			arrow = "→"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", diamond, arrow, j.packagePrependProjectName(pkg)))
		if pkgI != j.curPkgI {
			continue
		}
		for cmdIndex, cmd := range pkg.commands {
			arrow := " "
			if j.cmdCursor && j.cursorI == cmdIndex {
				arrow = "→"
			}
			circle := "○"
			if j.selected[j.curPkgI][cmdIndex] {
				circle = "●"
			}
			lines = append(lines, fmt.Sprintf("|  %s %s %s", arrow, circle, cmd.Desc))
		}
	}
	padLines := j.uiLines - len(lines) - 1
	if padLines < 1 {
		padLines = 1
	}
	for diff := padLines; diff > 0; diff-- {
		lines = append(lines, "|")
	}
	lines = append(lines, fmt.Sprintf("└ Press space to select, enter to continue. (%d/%d)", j.curPkgI+1, len(j.packages)))
	for _, line := range lines {
		fmt.Println(line)
	}
	if len(lines) >= j.uiLines {
		j.uiLines = len(lines)
	}
	go j.readKeyCmd()
}

func (j *ComposeProjectJob) packagePrependProjectName(pkg *Package) string {
	return filepath.Join(filepath.Base(j.cfg.Dir), pkg.name)
}

func (j *ComposeProjectJob) readKeyCmd() {
	if keyCmd, err := readKeyCmd(composeMenuKeyCmds); err != nil {
		j.doneC <- err
	} else {
		switch keyCmd {
		case Space:
			j.handleSpaceKey()
		case Enter:
			j.handleEnterKey()
		case Up:
			j.moveCursorUp()
		case Down:
			j.moveCursorDown()
		}
	}
}

func (j *ComposeProjectJob) moveCursorUp() {
	if j.cmdCursor {
		if j.cursorI == 0 {
			if j.curPkgI == 0 {
				go j.readKeyCmd()
			} else {
				j.cmdCursor = false
				j.cursorI = j.curPkgI - 1
				j.refreshInterface()
			}
		} else {
			j.cursorI--
			j.refreshInterface()
		}
	} else {
		if j.cursorI == 0 {
			go j.readKeyCmd()
		} else {
			j.cursorI--
			j.refreshInterface()
		}
	}
}

func (j *ComposeProjectJob) moveCursorDown() {
	if j.cmdCursor {
		if j.cursorI == len(j.packages[j.curPkgI].commands)-1 {
			go j.readKeyCmd()
		} else {
			j.cursorI++
			j.refreshInterface()
		}
	} else {
		if j.cursorI == j.curPkgI-1 {
			j.cmdCursor = true
			j.cursorI = 0
		} else {
			j.cursorI++
		}
		j.refreshInterface()
	}
}

func (j *ComposeProjectJob) handleEnterKey() {
	if j.cmdCursor {
		if j.curPkgI == len(j.packages)-1 {
			j.completeJob()
		} else {
			j.curPkgI++
			j.cursorI = 0
			j.refreshInterface()
		}
	} else {
		j.cmdCursor = true
		j.curPkgI = j.cursorI
		j.cursorI = 0
		j.refreshInterface()
	}
}

func (j *ComposeProjectJob) handleSpaceKey() {
	if j.cmdCursor {
		j.selected[j.curPkgI][j.cursorI] = !j.selected[j.curPkgI][j.cursorI]
		j.refreshInterface()
	} else {
		j.handleEnterKey()
	}
}

func (j *ComposeProjectJob) completeJob() {
	result := j.collectSelectedCommands()
	util.ClearTermLines(j.uiLines + 2)
	if len(result) == 0 {
		fmt.Println("┌ You did not select any commands!\n|")
		fmt.Println("└ Maestro is exiting without making any changes.")
	} else {
		up := NewUnicodePrinting()
		fmt.Printf("┌ Selected %d %s for Maestro to orchestrate:\n|\n", len(result), util.PluralPrint("package", len(result)))
		for _, pkg := range result {
			fmt.Printf("|   %s\n", j.packagePrependProjectName(pkg))
			for _, cmd := range pkg.commands {
				fmt.Printf("|    %s %s\n", up.greenCheck, cmd.Exec.ToString())
			}
		}

		j.cfg.AddPackages(result)
		if err := j.cfg.SaveConfig(); err != nil {
			log.Fatalln(err)
		}

		fmt.Print("|\n└ This composition was written to ./maestro.yml. Run `maestro` to continue.\n")
	}
	j.doneC <- nil
}

func (j *ComposeProjectJob) collectSelectedCommands() []*Package {
	var result []*Package
	for pkgI, pkg := range j.packages {
		var commands []*Command
		for cmdI, selected := range j.selected[pkgI] {
			if selected {
				cmd := pkg.commands[cmdI]
				commands = append(commands, &Command{
					Archetype: cmd.Archetype,
					Desc:      "",
					Dir:       cmd.Dir,
					Exec:      cmd.Exec,
					File:      cmd.File,
					Id:        cmd.Id,
					Name:      cmd.Name,
				})
			}
		}
		if len(commands) > 0 {
			result = append(result, &Package{
				commands: commands,
				dir:      pkg.dir,
				name:     pkg.name,
			})
		}
	}
	return result
}
