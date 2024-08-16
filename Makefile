
generate:
	protoc -I ../googleapis -I ../phantomapi --go_out=./api --go_opt=paths=source_relative ../phantom/api/v1/*

TOKEN := $(shell gcloud auth print-access-token )
SHA := $(shell git rev-parse HEAD)
deploy_production:
	phantom applications deploy -a panther -e production -c $(SHA) -t $(TOKEN)
