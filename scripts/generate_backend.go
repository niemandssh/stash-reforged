//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type loaderSpec struct {
	name string
	key  string
	val  string
}

var dataloaderSpecs = []loaderSpec{
	{"SceneLoader", "int", "*github.com/stashapp/stash/pkg/models.Scene"},
	{"GalleryLoader", "int", "*github.com/stashapp/stash/pkg/models.Gallery"},
	{"ImageLoader", "int", "*github.com/stashapp/stash/pkg/models.Image"},
	{"PerformerLoader", "int", "*github.com/stashapp/stash/pkg/models.Performer"},
	{"StudioLoader", "int", "*github.com/stashapp/stash/pkg/models.Studio"},
	{"TagLoader", "int", "*github.com/stashapp/stash/pkg/models.Tag"},
	{"GroupLoader", "int", "*github.com/stashapp/stash/pkg/models.Group"},
	{"FileLoader", "github.com/stashapp/stash/pkg/models.FileID", "github.com/stashapp/stash/pkg/models.File"},
	{"FolderLoader", "github.com/stashapp/stash/pkg/models.FolderID", "*github.com/stashapp/stash/pkg/models.Folder"},
	{"SceneFileIDsLoader", "int", "[]github.com/stashapp/stash/pkg/models.FileID"},
	{"ImageFileIDsLoader", "int", "[]github.com/stashapp/stash/pkg/models.FileID"},
	{"GalleryFileIDsLoader", "int", "[]github.com/stashapp/stash/pkg/models.FileID"},
	{"CustomFieldsLoader", "int", "github.com/stashapp/stash/pkg/models.CustomFieldMap"},
	{"SceneOCountLoader", "int", "int"},
	{"ScenePlayCountLoader", "int", "int"},
	{"SceneOHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"ScenePlayHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"SceneLastPlayedLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimePtr"},
	{"SceneWatchLaterLoader", "int", "bool"},
	{"SceneWatchLaterAtLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimePtr"},
	{"GalleryOCountLoader", "int", "int"},
	{"GalleryOHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"GalleryPlayCountLoader", "int", "int"},
	{"GalleryViewHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"GameLoader", "int", "*github.com/stashapp/stash/pkg/models.Game"},
	{"GameOCountLoader", "int", "int"},
	{"GameOHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"GameOMGHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
	{"GameViewCountLoader", "int", "int"},
	{"GameViewHistoryLoader", "int", "github.com/stashapp/stash/internal/api/loaders.TimeSlice"},
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "generate: unable to resolve repo root")
		os.Exit(1)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func ensureTool(root, name, pkg string) string {
	binPath := filepath.Join(root, "bin", name)
	modPath := filepath.Join(root, "go.mod")

	if stat, err := os.Stat(binPath); err == nil {
		if modStat, err := os.Stat(modPath); err == nil && stat.ModTime().After(modStat.ModTime()) {
			return binPath
		}
	}

	fmt.Fprintf(os.Stderr, "generate: building %s\n", name)
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("go", "build", "-o", binPath, pkg)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}

	return binPath
}

func runCmd(label string, dir string, name string, args ...string) error {
	fmt.Fprintf(os.Stderr, "generate: %s\n", label)
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGqlgen(root, gqlgen string) error {
	return runCmd("gqlgen (backend)", root, gqlgen)
}

func runDataloaders(root, dataloaden string) error {
	loadersDir := filepath.Join(root, "internal", "api", "loaders")
	fmt.Fprintf(os.Stderr, "generate: dataloaders (%d workers)\n", runtime.NumCPU())

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		first   error
		sem     = make(chan struct{}, runtime.NumCPU())
	)

	recordErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { first = err })
		}
	}

	for _, spec := range dataloaderSpecs {
		wg.Add(1)
		go func(spec loaderSpec) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cmd := exec.Command(dataloaden, spec.name, spec.key, spec.val)
			cmd.Dir = loadersDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			recordErr(cmd.Run())
		}(spec)
	}

	wg.Wait()
	return first
}

func runBackend(root string) error {
	gqlgen := ensureTool(root, "gqlgen", "github.com/99designs/gqlgen")
	dataloaden := ensureTool(root, "dataloaden", "github.com/vektah/dataloaden")

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		first   error
	)

	recordErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { first = err })
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		recordErr(runGqlgen(root, gqlgen))
	}()
	go func() {
		defer wg.Done()
		recordErr(runDataloaders(root, dataloaden))
	}()
	wg.Wait()

	return first
}

func runUIGqlgen(root string) error {
	uiDir := filepath.Join(root, "ui", "v2.5")
	return runCmd("gqlgen (ui)", uiDir, "bun", "run", "gqlgen")
}

func runLoginLocales(root string) error {
	uiDir := filepath.Join(root, "ui")
	return runCmd("login locales", uiDir, "go", "run", "-tags=dev", "../scripts/generateLoginLocales.go")
}

func runAll(root string) error {
	gqlgen := ensureTool(root, "gqlgen", "github.com/99designs/gqlgen")
	dataloaden := ensureTool(root, "dataloaden", "github.com/vektah/dataloaden")

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		first   error
	)

	recordErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { first = err })
		}
	}

	wg.Add(4)
	go func() {
		defer wg.Done()
		recordErr(runGqlgen(root, gqlgen))
	}()
	go func() {
		defer wg.Done()
		recordErr(runDataloaders(root, dataloaden))
	}()
	go func() {
		defer wg.Done()
		recordErr(runUIGqlgen(root))
	}()
	go func() {
		defer wg.Done()
		recordErr(runLoginLocales(root))
	}()
	wg.Wait()

	return first
}

func main() {
	all := flag.Bool("all", false, "regenerate backend, UI GraphQL, and login locales in parallel")
	uiOnly := flag.Bool("ui", false, "regenerate UI GraphQL only")
	loginOnly := flag.Bool("login-locale", false, "regenerate login locales only")
	dataloadersOnly := flag.Bool("dataloaders", false, "regenerate dataloaders only")
	flag.Parse()

	root := repoRoot()

	var err error
	switch {
	case *all:
		err = runAll(root)
	case *uiOnly:
		err = runUIGqlgen(root)
	case *loginOnly:
		err = runLoginLocales(root)
	case *dataloadersOnly:
		dataloaden := ensureTool(root, "dataloaden", "github.com/vektah/dataloaden")
		err = runDataloaders(root, dataloaden)
	default:
		err = runBackend(root)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}
