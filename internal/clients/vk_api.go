package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/config"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/models"
	"github.com/sirupsen/logrus"
)

type VKClient struct {
	AccessToken string
	BaseURL     string
	Client      *http.Client
}

func NewVKClient(accessToken string) *VKClient {
	return &VKClient{
		AccessToken: accessToken,
		BaseURL:     "https://api.vk.com/method/",
		Client:      &http.Client{},
	}
}

type VKError struct {
	Error struct {
		ErrorCode     int    `json:"error_code"`
		ErrorMsg      string `json:"error_msg"`
		RequestParams []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"request_params"`
	} `json:"error"`
}

// makeVKRequest выполняет GET-запрос к VK API и декодирует ответ.
func (vk *VKClient) makeVKRequest(ctx context.Context, method string, params url.Values, response interface{}) error {
	params.Set("access_token", vk.AccessToken)
	params.Set("v", config.VKAPIVersion)

	fullURL := fmt.Sprintf("%s%s?%s", vk.BaseURL, method, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"method": method,
			"url":    fullURL,
			"error":  err,
		}).Error("Не удалось создать HTTP-запрос")
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := vk.Client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"method": method,
			"url":    fullURL,
			"error":  err,
		}).Error("Ошибка выполнения VK API запроса")
		return fmt.Errorf("vk api call: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithFields(logrus.Fields{
				"method": method,
				"url":    fullURL,
				"error":  err,
			}).Warning("Не удалось закрыть тело ответа")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logrus.WithFields(logrus.Fields{
			"method":      method,
			"url":         fullURL,
			"status_code": resp.StatusCode,
			"body":        string(bodyBytes),
		}).Error("Неправильный статус код от VK API")
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		logrus.WithFields(logrus.Fields{
			"method": method,
			"url":    fullURL,
			"error":  err,
		}).Error("Ошибка декодирования JSON ответа от VK API")
		return fmt.Errorf("json decode: %w", err)
	}

	if vkErr, ok := response.(*VKError); ok && vkErr.Error.ErrorCode != 0 {
		logrus.WithFields(logrus.Fields{
			"method":     method,
			"url":        fullURL,
			"error_code": vkErr.Error.ErrorCode,
			"error_msg":  vkErr.Error.ErrorMsg,
		}).Error("VK API вернул ошибку")
		return fmt.Errorf("vk api error %d: %s", vkErr.Error.ErrorCode, vkErr.Error.ErrorMsg)
	}
	return nil
}

// GetCurrentUserID возвращает ID текущего пользователя.
func (vk *VKClient) GetCurrentUserID(ctx context.Context) (string, error) {
	params := url.Values{}

	type UsersGetResponse struct {
		Response []struct {
			ID int `json:"id"`
		} `json:"response"`
		Error VKError `json:"error"`
	}

	var response UsersGetResponse

	err := vk.makeVKRequest(ctx, "users.get", params, &response)
	if err != nil {
		return "", err
	}

	if len(response.Response) == 0 {
		return "", fmt.Errorf("empty response")
	}

	userID := strconv.Itoa(response.Response[0].ID)
	logrus.Infof("Получен ID текущего пользователя: %s", userID)
	return userID, nil
}

// CollectData собирает данные о пользователе, его фолловерах и подписках до заданной глубины.
func (vk *VKClient) CollectData(ctx context.Context, userID string, depth int) (*models.Data, error) {
	logrus.Infof("Начало сбора данных для пользователя ID: %s с глубиной: %d", userID, depth)
	data := &models.Data{
		Users:         make(map[int]models.User),
		Groups:        make(map[int]models.Group),
		Relationships: []models.Relationship{},
	}
	visitedUsers := make(map[int]bool)
	id, err := strconv.Atoi(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid userID: %w", err)
	}
	err = vk.collectUserData(ctx, id, data, visitedUsers, depth)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Сбор данных для пользователя ID: %s завершен успешно", userID)
	return data, nil
}

// collectUserData рекурсивно собирает данные пользователя.
func (vk *VKClient) collectUserData(ctx context.Context, userID int, data *models.Data, visitedUsers map[int]bool, depth int) error {
	if depth == 0 {
		return nil
	}
	if visitedUsers[userID] {
		return nil
	}
	visitedUsers[userID] = true

	// Получаем информацию о пользователе
	userInfo, err := vk.GetUserFullData(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка получения данных пользователя ID: %d: %v", userID, err)
		return fmt.Errorf("get user info (%d): %v", userID, err)
	}
	data.Users[userID] = userInfo

	// Получаем фолловеров
	followers, err := vk.GetFollowers(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка получения фолловеров пользователя ID: %d: %v", userID, err)
		return fmt.Errorf("get user followers (%d): %v", userID, err)
	}

	// Получаем подписки
	subscriptions, err := vk.GetSubscriptions(ctx, userID)
	if err != nil {
		logrus.Errorf("Ошибка получения подписок пользователя ID: %d: %v", userID, err)
		return fmt.Errorf("get user subscriptions (%d): %v", userID, err)
	}

	// Обработка фолловеров
	for _, follower := range followers {
		data.Relationships = append(data.Relationships, models.Relationship{
			From: follower.ID,
			To:   userID,
			Type: "FOLLOWS",
		})

		// Рекурсивный вызов для фолловера
		err = vk.collectUserData(ctx, follower.ID, data, visitedUsers, depth-1)
		if err != nil {
			logrus.Errorf("Ошибка рекурсивного сбора данных для пользователя ID: %d: %v", follower.ID, err)
			// Продолжаем сбор данных для остальных фолловеров
		}
	}

	// Обработка подписок
	for _, subscription := range subscriptions {
		if strings.ToLower(subscription.Type) == "page" || strings.ToLower(subscription.Type) == "group" {
			groupID := -subscription.ID
			data.Groups[groupID] = models.Group{
				ID:         subscription.ID,
				Name:       subscription.Name,
				ScreenName: subscription.ScreenName,
			}

			data.Relationships = append(data.Relationships, models.Relationship{
				From: userID,
				To:   groupID,
				Type: "SUBSCRIBES",
			})
		} else if strings.ToLower(subscription.Type) == "profile" {
			data.Relationships = append(data.Relationships, models.Relationship{
				From: userID,
				To:   subscription.ID,
				Type: "SUBSCRIBES",
			})

			err = vk.collectUserData(ctx, subscription.ID, data, visitedUsers, depth-1)
			if err != nil {
				logrus.Errorf("Ошибка рекурсивного сбора данных для подписки пользователя ID: %d: %v", subscription.ID, err)
				// Продолжаем сбор данных для остальных подписок
			}
		}
	}

	return nil
}

// GetUserFullData возвращает полные данные пользователя.
func (vk *VKClient) GetUserFullData(ctx context.Context, userID int) (models.User, error) {
	params := url.Values{}
	params.Set("user_ids", strconv.Itoa(userID))
	params.Set("fields", "followers_count,city,home_town,sex,screen_name")

	// Определяем структуру ответа
	type UserFullDataResponse struct {
		Response []struct {
			ID         int    `json:"id"`
			FirstName  string `json:"first_name"`
			LastName   string `json:"last_name"`
			ScreenName string `json:"screen_name"`
			Sex        int    `json:"sex"`
			City       struct {
				Title string `json:"title"`
			} `json:"city"`
			HomeTown string `json:"home_town"`
		} `json:"response"`
		Error VKError `json:"error"`
	}

	var response UserFullDataResponse

	// Выполняем запрос
	err := vk.makeVKRequest(ctx, "users.get", params, &response)
	if err != nil {
		return models.User{}, err
	}

	if len(response.Response) == 0 {
		logrus.Error("Получен пустой ответ от VK API в GetUserFullData")
		return models.User{}, fmt.Errorf("empty response")
	}

	user := response.Response[0]
	city := user.City.Title
	if city == "" {
		city = user.HomeTown
	}

	userModel := models.User{
		ID:         user.ID,
		ScreenName: user.ScreenName,
		Name:       fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		Sex:        user.Sex,
		City:       city,
	}

	return userModel, nil
}

// GetFollowers возвращает список фолловеров пользователя.
func (vk *VKClient) GetFollowers(ctx context.Context, userID int) ([]models.User, error) {
	params := url.Values{}
	params.Set("user_id", strconv.Itoa(userID))
	params.Set("fields", "screen_name,city,home_town,sex")
	params.Set("count", "200")

	// Определяем структуру ответа
	type GetFollowersResponse struct {
		Response struct {
			Items []struct {
				ID         int    `json:"id"`
				FirstName  string `json:"first_name"`
				LastName   string `json:"last_name"`
				ScreenName string `json:"screen_name"`
				Sex        int    `json:"sex"`
				City       struct {
					Title string `json:"title"`
				} `json:"city"`
				HomeTown string `json:"home_town"`
			} `json:"items"`
		} `json:"response"`
		Error VKError `json:"error"`
	}

	var response GetFollowersResponse

	// Выполняем запрос
	err := vk.makeVKRequest(ctx, "users.getFollowers", params, &response)
	if err != nil {
		return nil, err
	}

	followers := make([]models.User, len(response.Response.Items))
	for i, item := range response.Response.Items {
		city := item.City.Title
		if city == "" {
			city = item.HomeTown
		}
		followers[i] = models.User{
			ID:         item.ID,
			ScreenName: item.ScreenName,
			Name:       fmt.Sprintf("%s %s", item.FirstName, item.LastName),
			Sex:        item.Sex,
			City:       city,
		}
	}

	return followers, nil
}

// GetSubscriptions возвращает список подписок пользователя.
func (vk *VKClient) GetSubscriptions(ctx context.Context, userID int) ([]models.Subscription, error) {
	params := url.Values{}
	params.Set("user_id", strconv.Itoa(userID))
	params.Set("extended", "1")
	params.Set("count", "200")
	params.Set("fields", "screen_name,city,home_town,sex")

	// Определяем структуру ответа
	type GetSubscriptionsResponse struct {
		Response struct {
			Items []struct {
				ID         int    `json:"id"`
				Name       string `json:"name"`
				ScreenName string `json:"screen_name"`
				Type       string `json:"type"`
			} `json:"items"`
		} `json:"response"`
		Error VKError `json:"error"`
	}

	var response GetSubscriptionsResponse

	// Выполняем запрос
	err := vk.makeVKRequest(ctx, "users.getSubscriptions", params, &response)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]models.Subscription, len(response.Response.Items))
	for i, item := range response.Response.Items {
		subscriptions[i] = models.Subscription{
			ID:         item.ID,
			Name:       item.Name,
			ScreenName: item.ScreenName,
			Type:       item.Type,
		}
	}

	return subscriptions, nil
}
