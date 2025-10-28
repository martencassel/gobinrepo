package mw

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

const CorrelationIDHeader = "X-Correlation-ID"

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Ensure correlation ID
		corrID := c.GetHeader(CorrelationIDHeader)
		if corrID == "" {
			corrID = uuid.New().String()
		}
		// Add to context and response
		c.Set(CorrelationIDHeader, corrID)
		c.Writer.Header().Set(CorrelationIDHeader, corrID)

		// Wrap response writer
		blw := &bodyLogWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Compute digest of response body
		respDigest := digest.FromBytes(blw.body.Bytes())

		// Body size and duration
		size := blw.body.Len()
		duration := time.Since(start)

		// Pretty‑print JSON if applicable
		ct := c.Writer.Header().Get("Content-Type")
		if strings.Contains(ct, "application/json") {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, blw.body.Bytes(), "", "  "); err == nil {
				log.WithField("correlation_id", corrID).Debugf("Response JSON:\n%s", pretty.String())
			}
		}

		// Final summary log
		log.WithFields(log.Fields{
			"correlation_id": corrID,
			"method":         c.Request.Method,
			"path":           c.Request.URL.Path,
			"status":         c.Writer.Status(),
			"size_bytes":     size,
			"duration":       duration.Round(time.Millisecond),
			"digest":         respDigest,
		}).Info("Request handled")
	}
}
