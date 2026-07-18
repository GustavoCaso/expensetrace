package domain

type Banner struct {
	Icon    string
	Message string
}

type ViewBase struct {
	Error            string
	Banner           Banner
	CurrentPage      string
	LoggedIn         bool
	Username         string
	UsernameInitials string
}
