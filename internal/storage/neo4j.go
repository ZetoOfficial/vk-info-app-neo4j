package storage

import (
	"context"
	"fmt"
	"github.com/ZetoOfficial/vk-info-app-neo4j/internal/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
)

type Neo4jStorage struct {
	Driver neo4j.DriverWithContext
}

func NewNeo4jStorage(uri, username, password string) *Neo4jStorage {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		logrus.Fatalf("connect to driver: %v", err)
	}
	return &Neo4jStorage{Driver: driver}
}

func (s *Neo4jStorage) Close(ctx context.Context) error {
	return s.Driver.Close(ctx)
}

func (s *Neo4jStorage) SaveData(ctx context.Context, data *models.Data) error {
	session := s.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func(session neo4j.SessionWithContext, ctx context.Context) {
		err := session.Close(ctx)
		if err != nil {
			logrus.Warnf("close session: %v", err)
		}
	}(session, ctx)

	for _, user := range data.Users {
		_, err := session.Run(
			ctx,
			`
			MERGE (u:User {id: $id})
			SET u.name = $name, u.screen_name = $screen_name, u.sex = $sex, u.city = $city
			`,
			map[string]interface{}{
				"id":          user.ID,
				"name":        user.Name,
				"screen_name": user.ScreenName,
				"sex":         user.Sex,
				"city":        user.City,
			},
		)
		if err != nil {
			logrus.Errorf("save user %d: %v", user.ID, err)
			return fmt.Errorf("save user %d: %v", user.ID, err)
		}
	}

	for _, group := range data.Groups {
		_, err := session.Run(
			ctx,
			`
			MERGE (g:Group {id: $id})
			SET g.name = $name, g.screen_name = $screen_name
			`,
			map[string]interface{}{
				"id":          group.ID,
				"name":        group.Name,
				"screen_name": group.ScreenName,
			},
		)
		if err != nil {
			logrus.Errorf("save group %d: %v", group.ID, err)
			return fmt.Errorf("save group %d: %v", group.ID, err)
		}
	}

	for _, rel := range data.Relationships {
		fromLabel, toLabel := "User", "User"
		fromID, toID := rel.From, rel.To

		if rel.To < 0 {
			toLabel = "Group"
			toID = -rel.To
		}

		_, err := session.Run(
			ctx,
			fmt.Sprintf(`
			MATCH (from:%s {id: $from_id})
			MATCH (to:%s {id: $to_id})
			MERGE (from)-[:%s]->(to)
			`, fromLabel, toLabel, rel.Type),
			map[string]interface{}{
				"from_id": fromID,
				"to_id":   toID,
			},
		)
		if err != nil {
			logrus.Errorf("save relationship %s and %d и %d: %v", rel.Type, rel.From, rel.To, err)
			return fmt.Errorf("save relationship %s and %d и %d: %v", rel.Type, rel.From, rel.To, err)
		}
	}

	return nil
}

func (s *Neo4jStorage) RunQuery(ctx context.Context, queryName string) ([]map[string]interface{}, error) {
	query, exists := neo4jQueries[queryName]
	if !exists {
		return nil, fmt.Errorf("query %s not found", queryName)
	}

	session := s.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func(session neo4j.SessionWithContext, ctx context.Context) {
		err := session.Close(ctx)
		if err != nil {
			logrus.Warnf("close session: %v", err)
		}
	}(session, ctx)

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		recordMap := make(map[string]interface{})
		for _, key := range record.Keys {
			value, _ := record.Get(key)
			recordMap[key] = value
		}
		results = append(results, recordMap)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Neo4jStorage) Ping(ctx context.Context) error {
	session := s.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		if err := session.Close(ctx); err != nil {
			logrus.Warnf("close session: %v", err)
		}
	}()

	result, err := session.Run(ctx, "RETURN 1", nil)
	if err != nil {
		return fmt.Errorf("ping query failed: %w", err)
	}

	if result.Next(ctx) {
		return nil
	}
	if err = result.Err(); err != nil {
		return fmt.Errorf("ping query error: %w", err)
	}
	return fmt.Errorf("ping query did not return any results")
}
