package main

import (
	"fmt"
	"github.com/eighty4/maestro/git"
	"github.com/eighty4/maestro/util"
	"github.com/fatih/color"
	"log"
	"net/http"
	"os"
    "slices"
	"strconv"
)

const cmdMenuPreCompose = `Maestro --better-dx

  maestro git                sync a workspace of repositories
    --detail-local-changes   print excessive details about repos
    --offline                print repo statuses without syncing`

const cmdMenu = `Maestro --better-dx

  maestro                    start a project orchestration
    -c, --compose            compose a project orchestration
    -l, --ls                 list orchestrated commands

  maestro git                sync a workspace of repositories
    --detail-local-changes   print excessive details about repos
    --offline                print repo statuses without syncing`

func main() {
	if util.IsDebug() {
		color.HiYellow("debug enabled")
	}
	util.InitLogging()

	cfg, err := parseConfigFile(util.Cwd())
	if err != nil {
		println("config error:\n  " + err.Error())
		os.Exit(1)
	}

	if util.IsDebug() && cfg != nil {
		log.Printf("%s has %d %s", cfg.Filename, len(cfg.Repositories), util.SinglePrintIes("repositories", len(cfg.Repositories)))
	}

	if len(os.Args) > 1 && os.Args[1] == "git" {
		gitSync(cfg)
	} else {
		if util.IsFlagEnabled("MAESTRO_ORCHESTRATION") {
			orchestration(cfg)
		} else {
			fmt.Println(cmdMenuPreCompose)
		}
	}
}

func orchestration(cfg *Config) {
	color.HiYellow("orchestration feature flag enabled")
	if isArgSet("-c", "--compose") {
		if err := composeProject(cfg); err != nil {
			log.Fatalln(err)
		}
	} else if isArgSet("-l", "--ls") {
		lsCommands(cfg)
	} else if cfg.FileExists {
		if err := orchestrateProject(cfg); err != nil {
			log.Fatalln(err)
		}
	} else {
		fmt.Println(cmdMenu)
	}
}

func isArgSet(short string, long string) bool {
	for _, arg := range os.Args {
		if arg == short || arg == long {
			return true
		}
	}
	return false
}

func gitSyncOptions() *git.SyncOptions {
    validBoolFlags := []string{"true", "false", "1", "0"}
	syncOptions := &git.SyncOptions{}
    dlcEnvVar := os.Getenv("MAESTRO_DETAIL_LOCAL_CHANGES")
	if len(dlcEnvVar) > 0 {
        if !slices.Contains(validBoolFlags, dlcEnvVar) {
            fmt.Println("MAESTRO_DETAIL_LOCAL_CHANGES must be a true or false value")
            os.Exit(1)
        }
        syncOptions.DetailLocalChanges = dlcEnvVar == "true" || dlcEnvVar == "1"
	} else {
        syncOptions.DetailLocalChanges = slices.Contains(os.Args, "--detail-local-changes")
	}
    offlineEnvVar := os.Getenv("MAESTRO_OFFLINE")
    if len(offlineEnvVar) > 0 {
        if !slices.Contains(validBoolFlags, offlineEnvVar) {
            fmt.Println("MAESTRO_OFFLINE must be a true or false value")
            os.Exit(1)
        }
        syncOptions.Offline = offlineEnvVar == "true" || offlineEnvVar == "1"
    } else {
        syncOptions.Offline = slices.Contains(os.Args, "--offline")
    }
	return syncOptions
}

func gitSync(cfg *Config) {
	var repositories []*git.Repository
	if cfg != nil {
		repositories = cfg.Repositories
	}
	ws := git.NewWorkspace(cfg.Dir, repositories, 2)

	if len(ws.Repositories) == 0 {
		fmt.Println("No repositories found in this directory to sync.")
		os.Exit(1)
	}

	// calc max len of a repo name for print formatting
	maxNameLen := 0
	for _, repo := range ws.Repositories {
		nameLen := len(repo.Name)
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}

	up := NewUnicodePrinting()
	// escapes %% to % so given maxNameLen == 3, fmtStr := "  %3s %s\n"
	fmtStr := fmt.Sprintf("  %%%ds %%s %%s\n", maxNameLen)
	willNetworkError := !hasInternetConnection()
	syncErrorCount := 0

	printStatusUpdate := func(s *git.SyncUpdate) {
		var checkOrX string
		switch s.Status {
		case git.SyncSuccess:
			checkOrX = up.greenCheck
			break
		case git.SyncWarning:
			checkOrX = up.yellowCheck
			break
		case git.SyncFailure:
			checkOrX = up.redX
			syncErrorCount++
			break
		}
		if willNetworkError || s.Error == "" {
			// print status without error
			fmt.Printf(fmtStr, s.Repo, checkOrX, s.Message)
		} else {
			// print error with status
			fmt.Printf(fmtStr, s.Repo, checkOrX, s.Error)
			if len(s.Message) > 0 {
				fmt.Printf(fmtStr, "", up.yellowExclamation, s.Message)
			}
		}
	}

	println(fmt.Sprintf("Syncing %d repositories", len(ws.Repositories)))
    opts := gitSyncOptions()
	c := ws.Sync(opts)
	for {
		update, ok := <-c
		if ok {
			printStatusUpdate(update)
		} else {
            if opts.Offline {
                fmt.Printf("No repositories synced in %s.\n", color.New(color.Bold).Sprint("offline mode"))
            } else if !willNetworkError && syncErrorCount != len(ws.Repositories) {
				println("Done!")
			} else if willNetworkError {
                fmt.Printf("No repositories synced due to %s.\n", color.New(color.Bold).Sprint("network connectivity"))
			} else {
				println("No repositories were synced.")
			}
			break
		}
	}
}

func hasInternetConnection() bool {
	_, err := http.Get("https://github.com")
	return err == nil
}

type UnicodePrinting struct {
	greenCheck        string
	yellowCheck       string
	yellowExclamation string
	redX              string
}

func NewUnicodePrinting() UnicodePrinting {
	unquoteCodePoint := func(s string) string {
		result, err := strconv.Unquote(fmt.Sprintf("'%s'", s))
		if err != nil {
			log.Fatalln(err)
		}
		return result
	}
	check := unquoteCodePoint("\\U00002714")
	return UnicodePrinting{
		greenCheck:        color.New(color.FgGreen, color.Bold).Sprint(check),
		yellowCheck:       color.New(color.FgYellow, color.Bold).Sprint(check),
		yellowExclamation: color.New(color.FgYellow, color.Bold).Sprint("!"),
		redX:              color.New(color.FgRed, color.Bold).Sprint(unquoteCodePoint("\\U00002715")),
	}
}
