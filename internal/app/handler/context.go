package handler

type PageContext[T any] struct {
	Title string
	ActiveNav string
	IsLightTheme bool
	Data T
}

func NewPageContext[T any](title, activeNav string, isLightTheme bool, data T) PageContext[T] {
	return PageContext[T]{
		Title: title,
		ActiveNav: activeNav,
		IsLightTheme: isLightTheme,
		Data: data,
	}
}
