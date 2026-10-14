package moderation_api

import (
	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/daqing/airway/lib/render"
	"github.com/gin-gonic/gin"
)

// WordsAction reports the word list in use and where it came from, so an
// operator can confirm a reload landed.
func WordsAction(c *gin.Context) {
	render.OK(c, wordListPayload())
}

func wordListPayload() gin.H {
	service := moderation.Current()
	return gin.H{"categories": service.Categories(), "source": service.Info()}
}
