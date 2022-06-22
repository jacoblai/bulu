package model

type Config struct {
	Host      string   `json:"host"`
	PemPath   string   `json:"pemPath"`
	KeyPath   string   `json:"keyPath"`
	Proto     string   `json:"proto"`
	JwtSecret string   `json:"jwtSecret"`
	Domains   []Domain `json:"domains"`
	RateLimit Limit    `json:"rateLimit"`
}

type Limit struct {
	RateTime  string `json:"rateTime"`
	RateLimit int64  `json:"rateLimit"`
}

type Domain struct {
	Domain string `json:"domain"`
	Nodes  []Node `json:"nodes"`
}

type Node struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Weights uint32 `json:"weights"`
}
