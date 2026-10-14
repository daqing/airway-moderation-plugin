package moderation_api

import (
	"crypto/subtle"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecretHeader carries the shared secret callers present on every endpoint
// but health.
const SecretHeader = "X-Moderation-Secret"

// requireSecret guards an endpoint with the shared secret. The secret is
// unset on a development checkout, where the endpoints stay open; a host that
// runs one sets MODERATION_SECRET and every caller presents it.
func requireSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		want := strings.TrimSpace(os.Getenv("MODERATION_SECRET"))
		if want == "" {
			c.Next()
			return
		}
		if !secureEqual(c.GetHeader(SecretHeader), want) {
			fail(c, codeUnauthorized, "Invalid moderation secret")
			return
		}
		c.Next()
	}
}

// WarnUnprotected reports the open configuration at startup, so an unguarded
// host is not a silent surprise. The plugin root calls it when it boots.
func WarnUnprotected() {
	if strings.TrimSpace(os.Getenv("MODERATION_SECRET")) == "" {
		log.Print("moderation: MODERATION_SECRET is not set; screening endpoints are unprotected")
	}
}

func secureEqual(got, want string) bool {
	return len(got) == len(want) && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
