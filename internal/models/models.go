package models

// User представляет пользователя VK.
type User struct {
	ID         int
	ScreenName string
	Name       string
	Sex        int
	City       string
}

// Group представляет группу VK.
type Group struct {
	ID         int
	ScreenName string
	Name       string
}

// Subscription представляет подписку пользователя.
type Subscription struct {
	ID         int
	Name       string
	ScreenName string
	Type       string
}

// Relationship представляет связь между пользователями или группами.
type Relationship struct {
	From int
	To   int
	Type string
}

// Data представляет собранные данные.
type Data struct {
	Users         map[int]User
	Groups        map[int]Group
	Relationships []Relationship
}
