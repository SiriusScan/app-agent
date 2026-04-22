package bootstrap

import (
	"sync"

	"github.com/SiriusScan/app-agent/internal/commands"
	_ "github.com/SiriusScan/app-agent/internal/commands/help"
	_ "github.com/SiriusScan/app-agent/internal/commands/repo"
	_ "github.com/SiriusScan/app-agent/internal/commands/scan"
	_ "github.com/SiriusScan/app-agent/internal/commands/status"
	_ "github.com/SiriusScan/app-agent/internal/commands/sync"
	_ "github.com/SiriusScan/app-agent/internal/commands/template"
	_ "github.com/SiriusScan/app-agent/internal/commands/templatescan"
	_ "github.com/SiriusScan/app-agent/internal/modules/filecontent"
	_ "github.com/SiriusScan/app-agent/internal/modules/filehash"
	_ "github.com/SiriusScan/app-agent/internal/modules/filesearch"
	_ "github.com/SiriusScan/app-agent/internal/modules/versioncmd"
)

var bootstrapOnce sync.Once

func bootstrapCommands() {
	bootstrapOnce.Do(func() {
		commands.RegisterBuiltinAliases()
	})
}

func LoadCompatibilityRuntime() {
	bootstrapCommands()
}

func LoadConnectorRuntime() {
	bootstrapCommands()
}
