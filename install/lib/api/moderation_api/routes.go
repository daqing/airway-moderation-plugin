package moderation_api

import (
	"github.com/gin-gonic/gin"
)

// Routes registers the plugin's HTTP endpoints. The group is already mounted
// at the plugin's MountPath.
//
// Health stays open so a container probe needs no secret; everything that
// reads or serves the word list is guarded by the shared secret.
func Routes(r *gin.RouterGroup) {
	r.GET("/health", HealthAction)

	guarded := r.Group("", requireSecret())
	guarded.POST("/screen", ScreenAction)
	guarded.GET("/words", WordsAction)
	guarded.POST("/reload", ReloadAction)
}
