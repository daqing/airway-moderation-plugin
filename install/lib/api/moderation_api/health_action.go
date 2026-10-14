package moderation_api

import (
	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/daqing/airway/lib/render"
	"github.com/gin-gonic/gin"
)

// HealthAction answers an unauthenticated liveness probe. It reports the size
// and origin of the list in use — never the path or the words, which stay
// behind the secret — so a probe can tell a loaded list from the built-in
// fallback without leaking either.
func HealthAction(c *gin.Context) {
	source := moderation.Current().Info()
	origin := "embedded"
	if !source.Embedded {
		origin = "file"
	}
	render.OK(c, gin.H{
		"status":  "ok",
		"words":   source.Words,
		"source":  origin,
		"version": version,
	})
}
