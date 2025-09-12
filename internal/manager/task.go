package manager

import (
	"context"

	"github.com/stashapp/stash/pkg/job"
)

type Task interface {
	Start(context.Context)
	GetDescription() string
}

// TaskWithProgress is a task that supports progress reporting
type TaskWithProgress interface {
	StartWithProgress(context.Context, *job.Progress)
	GetDescription() string
}
