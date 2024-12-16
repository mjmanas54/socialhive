package models

type TemperaryUser struct {
	Otp  string `json:"otp" bson:"otp"`
	User User   `json:"user" bson:"user"`
}
