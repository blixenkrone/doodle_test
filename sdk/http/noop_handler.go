package znhttp

import (
	"net/http"

	"github.com/blixenkrone/sdk/logger"
)

type NoopHTTPHandler struct {
	logger         logger.Logger
	verboseLogging bool
}

// A handler to test or capture an HTTP endpoint route without executing any logic
func NewNoopHTTPHandler(
	logger logger.Logger,
	verboseLogging bool,
) NoopHTTPHandler {
	return NoopHTTPHandler{logger, verboseLogging}
}

func (n NoopHTTPHandler) Noop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if n.verboseLogging {
			n.logger.Debugf("request was made to %q", r.URL.String())
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
