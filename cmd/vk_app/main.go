package main

import (
	"context"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/app"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/cli"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/clients"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/config"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/logger"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/storage"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	err := godotenv.Load(config.DefaultEnvFile)
	if err != nil {
		logrus.Fatalf("load env: %v", err)
	}

	userID, logLevel, logFile, query := cli.ParseArgs()

	logger.Setup(logLevel, logFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vkClient := clients.NewVKClient(os.Getenv("VK_ACCESS_TOKEN"))
	neo4jStorage := storage.NewNeo4jStorage(
		os.Getenv("NEO4J_URI"),
		os.Getenv("NEO4J_USER"),
		os.Getenv("NEO4J_PASSWORD"),
	)
	if err := neo4jStorage.Ping(ctx); err != nil {
		logrus.Fatalf("Не удалось подключиться к Neo4j: %v", err)
	}
	logrus.Info("Подключение к Neo4j успешно установлено")
	defer func(neo4jStorage *storage.Neo4jStorage, ctx context.Context) {
		err := neo4jStorage.Close(ctx)
		if err != nil {
			logrus.Warningf("close neo4j storage: %v", err)
		}
	}(neo4jStorage, ctx)

	myApp := app.NewApp(vkClient, neo4jStorage)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logrus.Infof("Получен сигнал: %s. Завершение работы...", sig)
		cancel()
	}()

	if userID == "self" {
		resolvedID, err := vkClient.GetCurrentUserID(ctx)
		if err != nil {
			logrus.Fatalf("Ошибка получения текущего userID: %v", err)
		}
		userID = resolvedID
	}

	logrus.Infof("Resolved user ID: %s", userID)

	if err := myApp.Run(ctx, userID, 2, query); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Программа завершена успешно.")
}
