package ds

import (
	"errors"
	"time"
)

var ErrMigrationMethodNotFound = errors.New("migration method not found")

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

type MigrationMethod struct {
	ID          int64   `gorm:"primaryKey;column:id"`
	Title       string  `gorm:"column:title;type:varchar(255);uniqueIndex;not null"`
	Description *string `gorm:"column:description;type:text;"`
	Status      Status  `gorm:"column:status;type:varchar(20);not null;default:'draft'"`
	ImageURL    string  `gorm:"column:image_url;type:varchar(2048)"`
	VideoURL    string  `gorm:"column:video_url;type:varchar(2048)"`

	TimeInGb    *float64 `gorm:"column:time_in_gb;type:numeric(10,2)"`
	Reliability *float64 `gorm:"column:reliability;type:numeric(5,4)"`

	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	PublishedAt *time.Time `gorm:"column:published_at"`

	CreatorID int64 `gorm:"column:creator_id;not null"`
	Creator   User  `gorm:"foreignKey:CreatorID;constraint:OnDelete:RESTRICT"`
}

func (m *MigrationMethod) IsPublished() bool {
	return m.Status == StatusPublished
}

func (m *MigrationMethod) IsDeleted() bool {
	return m.Status == StatusDeleted
}

func (m *MigrationMethod) IsDraft() bool {
	return m.Status == StatusDraft
}

func (MigrationMethod) TableName() string {
	return "migration_methods"
}

type MigrationMethodLike struct {
	ID int64 `gorm:"primaryKey;column:id"`

	UserID int64 `gorm:"column:user_id;not null;uniqueIndex:idx_user_method"`
	User   User  `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT"`

	MethodID int64           `gorm:"column:method_id;not null;uniqueIndex:idx_user_method"`
	Method   MigrationMethod `gorm:"foreignKey:MethodID;constraint:OnDelete:RESTRICT"`
}

func (MigrationMethodLike) TableName() string {
	return "migration_method_likes"
}

type User struct {
	ID       int64  `gorm:"primaryKey;column:id"`
	Email    string `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	Password string `gorm:"column:password;type:varchar(255);not null"`
}

func (User) TableName() string {
	return "users"
}
