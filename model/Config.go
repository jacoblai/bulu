package model

type Config struct {
	Host      string `json:"host"`
	PemPath   string `json:"pemPath"`
	KeyPath   string `json:"keyPath"`
	Proto     string `json:"proto"`
	JwtSecret string `json:"jwtSecret"`
	Nodes     []Node `json:"nodes"`
}

type Node struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Weights uint32 `json:"weights"`
}
