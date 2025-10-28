package mw

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mgutz/ansi"
	log "github.com/sirupsen/logrus"
)

// flatten headers into "Key=Val1,Val2" and join with " | "
func formatHeaders(h map[string][]string) string {
	parts := make([]string, 0, len(h))
	for k, vals := range h {
		parts = append(parts, fmt.Sprintf("%s=%s", k, strings.Join(vals, ",")))
	}
	return strings.Join(parts, " | ")
}

// wrapper to count bytes written
type bodySizeWriter struct {
	gin.ResponseWriter
	size int
}

func (w *bodySizeWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func (w *bodySizeWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	w.size += n
	return n, err
}

func RequestTracer() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		cyan := ansi.ColorFunc("cyan")

		// Before proxying: request headers
		reqHeaders := formatHeaders(c.Request.Header)
		log.Debug(cyan(fmt.Sprintf("⇢ client %s %s | hdrs: %s", method, path, reqHeaders)))

		// Wrap writer to count body size
		bw := &bodySizeWriter{ResponseWriter: c.Writer}
		c.Writer = bw

		c.Next()

		// After response
		status := c.Writer.Status()
		dur := time.Since(start)
		respHeaders := formatHeaders(c.Writer.Header())

		bodyInfo := "body=empty"
		if bw.size > 0 {
			bodyInfo = fmt.Sprintf("body_size=%d", bw.size)
		}

		log.Debug(cyan(fmt.Sprintf("⇠ client %s %s | status=%d took=%s | %s | hdrs: %s",
			method, path, status, dur, bodyInfo, respHeaders)))
	}
}
