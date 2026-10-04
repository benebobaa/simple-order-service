// Command api runs the Simple Order Service HTTP server. All startup logic
// lives in internal/app so this entrypoint stays intentionally tiny.
package main

import (
	"log/slog"
	"os"

	"github.com/benebobaa/simple-order-service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}
