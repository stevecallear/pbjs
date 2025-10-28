.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative proto/pbjs/*.proto	
	protoc --go_out=. --go_opt=paths=source_relative internal/proto/testpb/*.proto	