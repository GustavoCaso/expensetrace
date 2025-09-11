package router

type banner struct {
	Icon    string
	Message string
}

type viewBase struct {
	Error  string
	Banner banner
}
