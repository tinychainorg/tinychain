package nakamoto

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/liamzebedee/tinychain-go/core/nakamoto/protobufs"
)

type ProtoNetPeer struct {
	pb.UnimplementedPeerServiceServer
}

func (p *ProtoNetPeer) Start() {
	port := 10000
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPeerServiceServer(grpcServer, p)
	grpcServer.Serve(lis)
}