package models

type Location struct {
	Latitude  *float64 `json:"lat" firestore:"lat" structs:"lat"`
	Longitude *float64 `json:"long" firestore:"long" structs:"long"`
}
