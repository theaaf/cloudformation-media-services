dist.zip: cloudformation-media-services
	zip -j dist.zip cloudformation-media-services

cloudformation-media-services: *.go
	GO111MODULE=on GOOS=linux go build .
