# Usage

```
curl --request POST \
  --url http://localhost:8080/lookup/ \
  --header 'Content-Type: application/json' \
  --data '{
	"query": ["mrkaran.dev"],
	"type": ["A"],
	"class": ["IN"],
	"nameservers": ["9.9.9.9"]
}'
```
