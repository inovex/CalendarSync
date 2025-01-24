package outlook_published

type Response struct {
	Body Body `json:"Body"`
}

type Body struct {
	ResponseMessages ResponseMessages `json:"ResponseMessages"`
}

type ResponseMessages struct {
	Items []Item `json:"Items"`
}

type Item struct {
	RootFolder RootFolder `json:"RootFolder"`
}

type RootFolder struct {
	Items []ItemRootFolder `json:"Items"`
}

type ItemRootFolder struct {
	Subject          string `json:"Subject"`
	Start            string `json:"Start"`
	End              string `json:"End"`
	CalendarItemType string `json:"CalendarItemType"`
	ItemId           ItemId `json:"ItemId"`
}

type ItemId struct {
	Id string `json:"Id"`
}
