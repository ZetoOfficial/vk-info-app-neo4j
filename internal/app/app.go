package app

import (
	"context"
	"fmt"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/models"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	SaveData(ctx context.Context, data *models.Data) error
	RunQuery(ctx context.Context, queryName string) ([]map[string]interface{}, error)
}

type VkApi interface {
	CollectData(ctx context.Context, userID string, depth int) (*models.Data, error)
}

type App struct {
	client  VkApi
	storage Storage
}

func NewApp(api VkApi, storage Storage) *App {
	return &App{api, storage}
}

func (a *App) Run(ctx context.Context, userID string, depth int, query string) error {
	if query != "" {
		logrus.Infof("Run query: %s", query)
		results, err := a.storage.RunQuery(ctx, query)
		if err != nil {
			return fmt.Errorf("run query: %v", err)
		}
		for _, result := range results {
			logrus.Info(result)
		}
		return nil
	}
	logrus.Info("Starting collect data")
	data, err := a.client.CollectData(ctx, userID, depth)
	if err != nil {
		return fmt.Errorf("collect data: %v", err)
	}
	logrus.Info("Save data to storage")
	err = a.storage.SaveData(ctx, data)
	if err != nil {
		return fmt.Errorf("save data: %v", err)
	}
	return nil
}
