package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ejholmes/slash"
	"golang.org/x/net/context"
)

func main() {
	h := slash.HandlerFunc(Handle)
	s := slash.NewServer(h)
	http.ListenAndServe(":8080", s)
}

func Handle(ctx context.Context, r slash.Responder, command slash.Command) error {
	if err := r.Respond(slash.Reply("Cool beans")); err != nil {
		return err
	}

	for i := 0; i < 4; i++ {
		<-time.After(time.Second)
		if err := r.Respond(slash.Reply(fmt.Sprintf("Async response %d", i))); err != nil {
			return err
		}
	}

	return nil
}

func printErrors(h slash.Handler) slash.Handler {
	return slash.HandlerFunc(func(ctx context.Context, r slash.Responder, command slash.Command) error {
		if err := h.ServeCommand(ctx, r, command); err != nil {
			fmt.Printf("error: %v\n", err)
		}
		return nil
	})
}
