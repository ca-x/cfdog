package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ServiceWeaver/weaver"
)

//go:generate weaver generate ./...

func main() {
	if err := weaver.Run(context.Background(), serve); err != nil {
		log.Fatal(err)
	}
}

// app implements the main component, the entry point to a Service Weaver app.
type app struct {
	weaver.Implements[weaver.Main]
	executor weaver.Ref[CloudflareUpdateExecutor]
	lis      weaver.Listener `weaver:"lis"`
}

// serve serves HTTP traffic.
func serve(ctx context.Context, app *app) error {
	executor := app.executor.Get()
	err := executor.Execute(ctx)
	if err != nil {
		app.Logger(ctx).Error("failed to execute executor", "error", err)
	}
	http.HandleFunc("/", helloHandler)
	app.Logger(ctx).Info("Listening on...", "address", app.lis)
	return http.Serve(app.lis, nil)
}

func helloHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "Hello, cloudflare dog🐶!")
}
