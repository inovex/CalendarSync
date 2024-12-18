package auth

type Credentials struct {
	Client Client
	Tenant Tenant
}

type Client struct {
	Id     string
	Secret string
}

type Tenant struct {
	Id string
}
