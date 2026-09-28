package serializer

import "github.com/Juuwe/data-migration-backend/internal/ds"

type MigrationMethod struct {
	ID                   int64    `json:"id"`
	Title                string   `json:"title"`
	Description          *string  `json:"description"`
	ImageURL             string   `json:"image_url"`
	VideoURL             string   `json:"video_url"`
	CreatedByCurrentUser bool     `json:"created_by_current_user"`
	LikesCount           int      `json:"likes_count"`
	TimeInGb             *float64 `json:"time_in_gb"`
	Reliability          *float64 `json:"reliability"`
}

func NewMethod(m ds.MigrationMethod, likesCount int, currentUserID int64, imageURL, videoURL string) MigrationMethod {
	return MigrationMethod{
		ID:                   m.ID,
		Title:                m.Title,
		Description:          m.Description,
		ImageURL:             imageURL,
		VideoURL:             videoURL,
		CreatedByCurrentUser: currentUserID > 0 && m.CreatorID == currentUserID,
		LikesCount:           likesCount,
		TimeInGb:             m.TimeInGb,
		Reliability:          m.Reliability,
	}
}

type MigrationMethodLike struct {
	ID         int64 `json:"id"`
	IsLiked    bool  `json:"is_liked"`
	LikesCount int   `json:"likes_count"`
}

func NewLike(methodID int64, isLiked bool, likesCount int) MigrationMethodLike {
	return MigrationMethodLike{ID: methodID, IsLiked: isLiked, LikesCount: likesCount}
}

type GridResponse struct {
	Methods []MigrationMethod `json:"methods"`
	MinTime float64           `json:"min_time"`
	MaxTime float64           `json:"max_time"`
}

type FeedResponse struct {
	Method MigrationMethod `json:"method"`
}

type DraftResponse struct {
	Draft MigrationMethod `json:"draft"`
}

type LikeResponse struct {
	Method MigrationMethodLike `json:"method"`
}

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

func NewUser(user ds.User) User {
	return User{ID: user.ID, Email: user.Email}
}
