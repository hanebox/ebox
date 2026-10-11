package providers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/contracts/queue"

	"ebox/app/facades"
	"ebox/app/services/backups"
	"ebox/app/services/crypto"
	"ebox/app/services/deployment"
	"ebox/app/services/events"
	"ebox/app/services/metrics"
	"ebox/app/services/notifications"
)

type EboxServiceProvider struct{}

func (p *EboxServiceProvider) Register(app foundation.Application) {
	if err := crypto.Init(facades.Config().GetString("app.key")); err != nil {
		panic(fmt.Sprintf("ebox: %s (set APP_KEY in your environment)", err))
	}

	ensureDataDirectories()

	metrics.Init(facades.Config().GetString("ebox.data_dir"))
}

func (p *EboxServiceProvider) Boot(app foundation.Application) {
	facades.Queue().Register([]queue.Job{
		&deployment.DeployApplicationJob{},
	})
}

func (p *EboxServiceProvider) Runners(app foundation.Application) []foundation.Runner {
	return []foundation.Runner{
		metrics.Default(),
		deployment.NewWorker(),
		events.NewRunner(),
		backups.NewRunner(),
		notifications.NewWatcher(),
	}
}

func ensureDataDirectories() {
	paths := []string{
		facades.Config().GetString("ebox.data_dir"),
		facades.Config().GetString("ebox.apps_dir"),
		facades.Config().GetString("ebox.volumes_dir"),
		facades.Config().GetString("ebox.backups_dir"),
	}

	for _, dir := range paths {
		if dir == "" {
			continue
		}

		if err := os.MkdirAll(filepath.Clean(dir), 0o755); err != nil {
			panic(fmt.Sprintf("ebox: failed to create data directory %s: %s", dir, err))
		}
	}
}
