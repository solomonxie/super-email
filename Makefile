export
AWS_REGION   := us-east-1
PROJECT_NAME := super-email
TF_DIR       := terraform/envs/prod


init:
	terraform -chdir=${TF_DIR} init

plan:
	terraform -chdir=${TF_DIR} plan

deploy:
	terraform -chdir=${TF_DIR} apply
	# also rebuilds + redeploys any Lambda whose Go source changed —
	# there's no separate "push code" step, terraform apply is it.

destroy:
	terraform -chdir=${TF_DIR} destroy


build:
	go build ./...

test:
	go test ./...

tidy:
	go mod tidy

# make logs-email-router / logs-digest-bible / ...
logs-%:
	aws logs tail /aws/lambda/${PROJECT_NAME}-$* --follow --region ${AWS_REGION}
