{
	"host": ":80",
	"pemPath": "server.pem",
	"keyPath": "server.key",
	"proto": "http",
	"jwtSecret": "none",
	"rateLimit": {
	"rateTime": "1s",
		"rateLimit": 150000
	},
	"domains": [
	{
		"domain": "aaa.xbyct.net",
		"nodes": [
			{
				"name": "node1",
				"url": "http://127.0.0.1:8001",
				"weights": 100
			},
			{
				"name": "node2",
				"url": "http://127.0.0.1:8002",
				"weights": 100
			}
		]
	},
	{
		"domain": "bbb.xbyct.net",
		"nodes": [
			{
				"name": "node3",
				"url": "http://127.0.0.1:8003",
				"weights": 100
			},
			{
				"name": "node4",
				"url": "http://127.0.0.1:8004",
				"weights": 100
			}
		]
	},
	{
		"domain": "ccc.xbyct.net",
		"nodes": [
			{
				"name": "node5",
				"url": "http://127.0.0.1:8005",
				"weights": 100
			},
			{
				"name": "node6",
				"url": "http://127.0.0.1:8006",
				"weights": 100
			}
		]
	}
]
}