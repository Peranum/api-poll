package httptools

import "fmt"

type Proxy struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Host     string `json:"host"     validate:"required"`
	Port     int    `json:"port"     validate:"required,gt=0,lt=65536"`
}

func (p Proxy) String() string {
	return fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.Host, p.Port)
}
