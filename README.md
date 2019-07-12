# Killbot Discord Integration

This integration runs and pulls kills as they happen on zkill. If the kill is associated with either the `ALLIANCE_ID` or `CORPORATION_ID` then it pushes a custom built attachment to the `WEBHOOK` uri.

Due to extremely high likely-hood that ESI returns a 503, it automatically retries 3 times before giving up. Discord also tends to rate limit the pushes when a huge amount of kills happen all at once. Since each kill is processed on a different goroutine, the process is paused until the timeout ends and continues trying until the post gets through.

## Build

All you really need to do is to provide the three environment variables (However you choose)

```
ALLIANCE_ID
CORPORATION_ID
WEBHOOK
```

Then just run the docker build like normal:

```
docker build -t killbot-integration .
```

## Deployment

Personally, I run this on my own Kubernetes cluster, so I have provided a deployment.yaml that works perfectly if you are doing the same.

### __Make sure you update the image url that is being pulled!__