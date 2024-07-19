set -ex

# protoc -I=. --go_out=. --go_opt=paths=source_relative core/nakamoto/proto/*.proto
# protoc -I=core/nakamoto/proto/ --go_out=core/nakamoto/proto --go_opt=paths=source_relative core/nakamoto/proto/*.proto

SRCDIR=core/nakamoto/protobufs
cd $SRCDIR
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative services.proto
