// Package moderationplugin implements the moderation Airway plugin: a word
// list service other services screen message content against.
package moderationplugin

import (
	"fmt"
	"log"

	"github.com/daqing/airway-moderation-plugin/install/lib/api/moderation_api"
	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/daqing/airway/lib/plugin"
	"github.com/gin-gonic/gin"
)

// Plugin is the moderation feature module.
type Plugin struct{}

func (Plugin) Name() string      { return "moderation" }
func (Plugin) MountPath() string { return "/api/v1/moderation" }

func (Plugin) Routes(r *gin.RouterGroup) {
	moderation_api.Routes(r)
}

// Boot loads the word list the service screens against. A list that is not
// there falls back to the built-in one, which keeps a checkout that never ran
// plugin:install working; a list that is there but unusable fails the boot,
// because screening against a list nobody chose is worse than not starting.
func (Plugin) Boot() error {
	path := moderation.WordsFile()
	if err := moderation.Setup(path); err != nil {
		return fmt.Errorf("moderation plugin: %w", err)
	}
	source := moderation.Current().Info()
	if source.Embedded {
		log.Printf("moderation: %s is not there; screening against the built-in word list (%d words)", path, source.Words)
	} else {
		log.Printf("moderation: screening against %s (%d words)", source.Path, source.Words)
	}
	moderation_api.WarnUnprotected()
	return nil
}

func init() {
	moderation_api.SetVersion(Version())
	plugin.Register(Plugin{})
}
