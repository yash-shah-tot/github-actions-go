package models

type GoogleTimeZone struct {
	DstOffset    float64 `json:"dstOffset"`
	RawOffset    float64 `json:"rawOffset"`
	Status       string  `json:"status"`
	TimezoneID   string  `json:"timeZoneID"`
	TimezoneName string  `json:"timeZoneName"`
}
