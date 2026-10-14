package moderation_api

import (
	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/daqing/airway/lib/render"
	"github.com/gin-gonic/gin"
)

// maxContentBytes caps what one screening request may carry. A message is
// small; anything larger is a caller bug or an attempt to make the service
// burn memory.
const maxContentBytes = 32 << 10

type screenRequest struct {
	Content string `json:"content"`
}

// ScreenAction reports the words the content hits. A caller that cannot reach
// this endpoint has to decide for itself what to do with the message; the
// rule the IM plugin follows is to hold it back rather than deliver it.
func ScreenAction(c *gin.Context) {
	var request screenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, codeInvalidRequest, "Invalid screening request")
		return
	}
	if len(request.Content) > maxContentBytes {
		fail(c, codeInvalidRequest, "Content is too long to screen")
		return
	}
	render.OK(c, moderation.Current().Screen(request.Content))
}
