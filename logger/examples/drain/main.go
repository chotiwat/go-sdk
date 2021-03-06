package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/blend/go-sdk/logger"
)

func main() {
	log := logger.MustNew(logger.OptAll())

	log.Listen(logger.Info, "randomly_slow", func(ctx context.Context, e logger.Event) {
		if rand.Float64() < 0.1 {
			fmt.Println("randomly slow start")
			time.Sleep(500 * time.Millisecond)
			fmt.Println("randomly slow stop")
		}
	})

	infoSignal := time.Tick(100 * time.Millisecond)

	done := time.After(10 * time.Second)

	for {
		select {
		case <-infoSignal:
			log.Infof("this is an info event")
		case <-done:
			fmt.Println("draining")
			log.Drain()
			fmt.Println("exiting")
			os.Exit(0)
		}
	}
}
