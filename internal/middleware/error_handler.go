package middleware

import (
	apiError "collaborative-markdown-editor/internal/errors"
	"errors"

	log "github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Execute the handler first

		// detect any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			var apiErr *apiError.APIError

			// if it's our custom APIError
			if !errors.As(err, &apiErr) {
				// If it's a raw error we didn't wrap, treat as Internal
				apiErr = apiError.Internal(err)
			}

			// LOGGING
			if apiErr.Status >= 500 {
				log.Error().Err(apiErr.Internal).Msg("internal error")
			} else {
				log.Info().Err(apiErr.Internal).Msg(apiErr.Message)
			}

			// Respond with JSON
			c.AbortWithStatusJSON(apiErr.Status, apiErr)
		}
	}
}
