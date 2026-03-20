package models

type UserID string

type AuthorID string

type ClickEvent struct {
	Author AuthorID
	User   UserID
}
