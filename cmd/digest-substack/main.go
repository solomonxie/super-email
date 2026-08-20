// Command digest-substack sends the Substack "old post" refresher
// digest. Triggered by an EventBridge Scheduler cron rule. See
// DESIGN.md. Placeholder until the feed/sitemap crawl lands.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) error {
	log.Printf("digest-substack: received event: %s", event)
	return nil
}

func main() {
	lambda.Start(handler)
}
