package model

type Status int

const (
	StatusDraft Status = iota
	StatusDeleted
	StatusPublished
)


type MigrationMethod struct {
	ID int
	Name string
	Title string
	Description string
	VideoKey string
	ImageKey string
	Status Status
	TimeInGb float64
	Reliability float64
	Likes []int
}

func (m *MigrationMethod) IsDeleted() bool {
	return m.Status == StatusDeleted
}

func (m *MigrationMethod) IsDraft() bool {
	return m.Status == StatusDraft
}

func (m *MigrationMethod) IsPublished() bool {
	return m.Status == StatusPublished
}
