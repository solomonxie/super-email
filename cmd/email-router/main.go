// Command email-router is invoked by the SES receipt rule for inbound
// mail. It will parse the raw MIME message SES drops in S3, verify the
// sender, route note commands, and reply — see DESIGN.md §3, §5.
// Placeholder until that logic lands.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) error {
	log.Printf("email-router: received event: %s", event)
	return nil
}

func main() {
	lambda.Start(handler)
}
