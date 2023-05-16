package auth

type Credentials struct {
	Client     Client
	Tenant     Tenant
	CalendarId string
}

type Client struct {
	Id     string
	Secret string
}

type Tenant struct {
	Id string
}
