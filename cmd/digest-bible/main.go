// Command digest-bible sends the daily Bible reading digest. Triggered
// by an EventBridge Scheduler cron rule. See DESIGN.md §6. Placeholder
// until the reading-plan and SES send logic land.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) error {
	log.Printf("digest-bible: received event: %s", event)
	return nil
}

func main() {
	lambda.Start(handler)
}
