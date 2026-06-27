package models

type SiteSetting struct {
	ID          uint   `gorm:"primaryKey"`
	SiteName    string `gorm:"default:Omoikane"`
	Tagline     string `gorm:"default:A headless-ish CMS"`
	Logo        string
	Favicon     string
	BlogEnabled bool `gorm:"default:true"`
}
