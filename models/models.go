package models


type Upload struct {
	Id string  `json:"Id" gorm:"primaryKey"`
	Url string  `json:"url"`
	FileType string  `json:"fileType"`
	CreatedAt int64  `json:"createdAt"`
	Size string `json:"size"`
	ApplicationId string `json:"applicationId"`
	UserId string `json:"userId"`
}

