package ds

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

type MigrationMethod struct {
	ID          int64  `gorm:"primaryKey;column:id"`
	Title       string `gorm:"column:title;type:varchar(255);not null"`
	Description string `gorm:"column:description;type:text;not null"`
	Status      Status `gorm:"column:status;type:varchar(20);not null;default:'draft'"`
	ImageKey    string `gorm:"column:image_key;type:varchar(512)"`
	VideoKey    string `gorm:"column:video_key;type:varchar(512)"`

	TimeInGb    float64 `gorm:"column:time_in_gb;not null"`
	Reliability float64 `gorm:"column:reliability;type:numeric(5,4);not null"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	FormedAt  time.Time `gorm:"column:formed_at;not null"`

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

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (MigrationMethodLike) TableName() string {
	return "migration_method_likes"
}

type User struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	Email     string    `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (User) TableName() string {
	return "users"
}
