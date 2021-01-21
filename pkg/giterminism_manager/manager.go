package giterminism_manager

import (
	"context"
	"fmt"

	"github.com/werf/werf/pkg/git_repo"
	"github.com/werf/werf/pkg/giterminism_manager/config"
	"github.com/werf/werf/pkg/giterminism_manager/file_reader"
	"github.com/werf/werf/pkg/giterminism_manager/inspector"
)

type NewManagerOptions struct {
	LooseGiterminism bool
	DevMode          bool
}

func NewManager(ctx context.Context, projectDir string, localGitRepo git_repo.Local, headCommit string, options NewManagerOptions) (Interface, error) {
	sharedContext, err := NewSharedContext(projectDir, localGitRepo, headCommit, NewSharedContextOptions{
		looseGiterminism: options.LooseGiterminism,
		devMode:          options.DevMode,
	})
	if err != nil {
		return nil, err
	}

	fr := file_reader.NewFileReader(sharedContext)

	c, err := config.NewConfig(ctx, fr)
	if err != nil {
		return nil, err
	}

	fr.SetGiterminismConfig(c)

	i := inspector.NewInspector(c, sharedContext)

	m := &Manager{
		sharedContext: sharedContext,
		fileReader:    fr,
		inspector:     i,
	}

	return m, nil
}

type Manager struct {
	fileReader FileReader
	inspector  Inspector

	*sharedContext
}

func (m Manager) FileReader() FileReader {
	return m.fileReader
}

func (m Manager) Inspector() Inspector {
	return m.inspector
}
