package main

import (
	"context"
	"delayedNotifier/internal/infrastructure/data"
	"delayedNotifier/internal/infrastructure/data/repositories"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wb-go/wbf/retry"
)

func main() {
	db, err := data.InitDb()
	if err != nil {
		log.Fatal(err)
	}
	notificationRepository := repositories.NewNotificationRepository(db, retry.Strategy{})
	_ = notificationRepository
	log.Println("server is running")
	go gracefulShutdown()
}

func gracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
}
