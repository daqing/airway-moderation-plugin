package moderation_api

import (
	"log"

	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/daqing/airway/lib/render"
	"github.com/gin-gonic/gin"
)

// ReloadAction re-reads the word list from disk, so an edit takes effect
// without a restart. A list that cannot be read or parsed is refused and the
// list already in use keeps screening.
func ReloadAction(c *gin.Context) {
	if err := moderation.Current().Reload(); err != nil {
		log.Printf("moderation: reload failed: %v", err)
		fail(c, codeWordList, "Could not load the word list: "+err.Error())
		return
	}
	render.OK(c, wordListPayload())
}
