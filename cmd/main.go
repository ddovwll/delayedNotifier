package main

import (
	"context"
	_ "delayedNotifier/docs"
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/infrastructure/cache"
	"delayedNotifier/internal/infrastructure/data"
	"delayedNotifier/internal/infrastructure/data/repositories"
	"delayedNotifier/internal/infrastructure/message_queue"
	"delayedNotifier/internal/infrastructure/notifier"
	"delayedNotifier/internal/infrastructure/notifier/notifier_telegram"
	retry2 "delayedNotifier/internal/infrastructure/retry"
	"delayedNotifier/internal/web_api/controllers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

// @title        Delayed Notifier API
// @version      1.0
// @description  HTTP API для управления отложенными уведомлениями (создание, проверка статуса и отмена рассылок).
// @BasePath     /
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	db, err := data.InitDb()
	if err != nil {
		log.Fatal(err)
	}
	notificationRepository := repositories.NewNotificationRepository(db, retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  1,
	})
	conn, err := rabbitmq.Connect(os.Getenv("RABBITMQ_URL"), 5, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	channel, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	publisher := rabbitmq.NewPublisher(channel, os.Getenv("RABBITMQ_EXCHANGE"))
	producer := message_queue.NewRabbitProducer(publisher, retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  1,
	})
	consConf := rabbitmq.NewConsumerConfig(os.Getenv("RABBITMQ_QUEUE"))
	cons := rabbitmq.NewConsumer(channel, consConf)
	deliveryTaskService := services.NewDeliveryTaskService(producer, os.Getenv("RABBITMQ_ROUTING_KEY"))
	telegramNotiifierrepository := notifier_telegram.NewTelegramNotifierRepository(db, retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  1,
	})
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	notifierFactory := notifier.NewNotifierFactory(bot, telegramNotiifierrepository)
	telegramListener := notifier_telegram.NewTelegramBotListener(bot, telegramNotiifierrepository)
	retryer := retry2.NewRetryer(retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  1,
	})
	redisClient := redis.New(os.Getenv("REDIS_ADDRESS"), os.Getenv("REDIS_PASSWORD"), 0)
	cache := cache.NewRedisCache(redisClient)
	notificationService := services.NewNotificationService(notificationRepository, deliveryTaskService, notifierFactory, retryer, cache)
	consumer := message_queue.NewRabbitConsumer(cons, notificationService, retry.Strategy{
		Attempts: 3,
		Delay:    1 * time.Second,
		Backoff:  1,
	}, channel)
	controller := controllers.NewNotificationController(notificationService)
	server := createServer(controller)
	log.Println("server is running")
	go gracefulShutdown(server, consumer, telegramListener, redisClient, channel, db)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func gracefulShutdown(server *http.Server, consumer *message_queue.RabbitConsumer, telegramListener *notifier_telegram.TelegramBotListener, redis *redis.Client, channel *amqp091.Channel, db *dbpg.DB) {
	defer func() {
		err := redis.Close()
		if err != nil {
			log.Println(err)
		}
		err = channel.Close()
		if err != nil {
			log.Println(err)
		}
		err = db.Master.Close()
		if err != nil {
			log.Println(err)
		}
		for _, slave := range db.Slaves {
			err := slave.Close()
			if err != nil {
				log.Println(err)

			}
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go consumer.StartConsumer(ctx, 12)
	go telegramListener.SetupBot(ctx)
	<-c
	servCtx, cancelServ := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelServ()
	cancel()
	err := server.Shutdown(servCtx)
	if err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
}

func createServer(controller *controllers.NotificationController) *http.Server {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr: ":" + os.Getenv("HTTP_PORT"),
	}

	controller.MapRoutes(mux)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	server.Handler = mux
	return server
}
