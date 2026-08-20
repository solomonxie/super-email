// Command digest-youtube sends the daily YouTube uploads digest.
// Triggered by an EventBridge Scheduler cron rule. See DESIGN.md.
// Placeholder until the YouTube Data API integration lands.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) error {
	log.Printf("digest-youtube: received event: %s", event)
	return nil
}

func main() {
	lambda.Start(handler)
}
