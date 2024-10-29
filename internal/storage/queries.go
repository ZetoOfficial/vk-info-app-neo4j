package storage

var neo4jQueries = map[string]string{
	// всего пользователей
	"total_users": `
			MATCH (u:User)
			RETURN COUNT(u) AS total_users
		`,
	// всего групп
	"total_groups": `
			MATCH (g:Group)
			RETURN COUNT(g) AS total_groups
		`,
	// топ 5 пользователей по количеству фоллоуверов
	"top_users": `
			MATCH (u:User)<-[:FOLLOWS]-(f:User)
			RETURN u.id AS user_id, COUNT(f) AS followers_count
			ORDER BY followers_count DESC
			LIMIT 5
		`,
	//  топ 5 самых популярных групп
	"top_groups": `
			MATCH (g:Group)<-[:SUBSCRIBES]-(u:User)
			RETURN g.name AS group_name, COUNT(u) AS subscribers_count
			ORDER BY subscribers_count DESC
			LIMIT 5
		`,
	//  все пользователи, которые фолоуверы друг друга.
	"mutual_followers": `
			MATCH (u1:User)-[:FOLLOWS]->(u2:User)
			MATCH (u2)-[:FOLLOWS]->(u1)
			RETURN u1.id AS user1_id, u2.id AS user2_id
		`,
	// Топ-5 пользователей по количеству подписок на группы
	"top_subscribers": `
		MATCH (u:User)-[:SUBSCRIBES]->(g:Group)
		RETURN u.id AS user_id, COUNT(g) AS subscription_count
		ORDER BY subscription_count DESC
		LIMIT 5
	`,
	// Топ-5 популярных городов среди пользователей
	"top_cities": `
		MATCH (u:User)
		WHERE u.city IS NOT NULL
		RETURN u.city AS city, COUNT(u) AS user_count
		ORDER BY user_count DESC
		LIMIT 5
	`,
	// Пользователи с наибольшим количеством общих подписчиков с другими пользователями
	"top_mutual_followers": `
		MATCH (u1:User)<-[:FOLLOWS]-(mutualFollower)-[:FOLLOWS]->(u2:User)
		WHERE u1 <> u2
		WITH u1, COUNT(DISTINCT mutualFollower) AS mutual_followers_count
		RETURN u1.id AS user_id, mutual_followers_count
		ORDER BY mutual_followers_count DESC
		LIMIT 5
	`,
}
