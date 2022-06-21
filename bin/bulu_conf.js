{
	"host": ":7003",
	"pemPath": "server.pem",
	"keyPath": "server.key",
	"proto": "http",
	"nodes": [
		{
			"name": "node1",
			"url": "http://127.0.0.1:7001/",
			"weights": 100
		},
		{
			"name": "node2",
			"url": "http://127.0.0.1:7002/",
			"weights": 100
		},
		{
			"name": "node3",
			"url": "http://127.0.0.1:7004/",
			"weights": 100
		},
		{
			"name": "node4",
			"url": "http://127.0.0.1:7005/",
			"weights": 100
		},
		{
			"name": "node5",
			"url": "http://127.0.0.1:7006/",
			"weights": 100
		}
	]
}