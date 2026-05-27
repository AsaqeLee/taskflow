package model

type User struct {
	ID    string `json:"id" bson:"_id"`
	Name  string `json:"name" bson:"name"`
	Role  string `json:"role" bson:"role"`
	Token string `json:"token" bson:"token"`
}
