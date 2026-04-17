// Command digest-social sends the daily Facebook/Instagram digest.
// Triggered by an EventBridge Scheduler cron rule. See DESIGN.md §6.
// Placeholder until the Meta Graph API integration lands.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) error {
	log.Printf("digest-social: received event: %s", event)
	return nil
}

func main() {
	lambda.Start(handler)
}
