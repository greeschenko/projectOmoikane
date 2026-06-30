package models

type SiteSetting struct {
	ID                uint   `gorm:"primaryKey"`
	SiteName          string `gorm:"default:Omoikane"`
	Tagline           string `gorm:"default:A headless-ish CMS"`
	Logo              string
	Favicon           string
	BlogEnabled       bool   `gorm:"default:true"`
	ResetEmailSubject string `gorm:"default:Password Reset Request"`
	ResetEmailBodyHTML string `gorm:"type:text;default:<h2>Password Reset</h2><p>Click <a href=\"{{.ResetLink}}\">here</a> to reset your password. Expires in {{.ExpiryHours}} hour(s).</p><p>If you did not request this, ignore this email.</p>"`
}
