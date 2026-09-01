// Package models contains all domain data models
package models

type Task struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
