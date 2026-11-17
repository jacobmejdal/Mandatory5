package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/jacobmejdal/Mandatory5/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	id := os.Args[1] // "0" eller "1"

	port, peerAddr, primary := "50051", "localhost:50052", true
	if id == "1" {
		port, peerAddr, primary = "50052", "localhost:50051", false
	}

	// forbindelse til den anden node
	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to peer: %v", err)
	}

	s := &Server{
		bids:           make(map[int32]int32),
		ids:            make(map[int32]int64),
		auctionEndTime: time.Now().Add(100 * time.Second),
		peer:           pb.NewAuctionClient(conn),
		isPrimary:      primary,
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterAuctionServer(grpcServer, s)
	log.Println("server listening on :" + port)
	if !s.isPrimary {
		go s.watchPrimary()
	}
	grpcServer.Serve(lis)
}

type Server struct {
	pb.UnimplementedAuctionServer

	mu             sync.Mutex
	bids           map[int32]int32
	ids            map[int32]int64
	highestBid     int32
	highestBidder  int32
	auctionEndTime time.Time
	isPrimary      bool // er jeg primary?
	peer           pb.AuctionClient
}

func (s *Server) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().After(s.auctionEndTime) {
		// auktionen er slut
		return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION}, nil
	}

	// 2. Gentaget bud?
	lastId, ok := s.ids[req.Bidder]
	if ok && lastId == req.Id {
		return &pb.BidReply{Outcome: pb.BidReply_SUCCESS}, nil
	}

	// 3. For lavt bud?
	if req.Amount <= s.highestBid {
		return &pb.BidReply{Outcome: pb.BidReply_FAIL}, nil
	}

	// 4. Opdater højeste bud
	s.highestBid = req.Amount
	s.highestBidder = req.Bidder
	s.ids[req.Bidder] = req.Id

	// 5. send til backuppen, hvis jeg er primary
	if s.isPrimary {
		// send til backuppen
		_, err := s.peer.Update(ctx, req)
		if err != nil {
			log.Printf("Error updating backup: %v", err)
		}
	}
	return &pb.BidReply{Outcome: pb.BidReply_SUCCESS}, nil
}

func (s *Server) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return &pb.ResultReply{
		HighestBidder: s.highestBidder,
		Amount:        s.highestBid,
		AuctionOver:   time.Now().After(s.auctionEndTime),
	}, nil

}

func (s *Server) Heartbeat(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil

}

func (s *Server) Update(ctx context.Context, req *pb.BidRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.highestBid = req.Amount
	s.highestBidder = req.Bidder
	s.ids[req.Bidder] = req.Id

	return &pb.Empty{}, nil
}

func (s *Server) watchPrimary() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := s.peer.Heartbeat(ctx, &pb.Empty{})
		cancel()

		if err != nil {
			log.Println("primary is dead, taking over")
			s.mu.Lock()
			s.isPrimary = true
			s.mu.Unlock()
			return // stop løkken, nu er jeg primary
		}
	}
}
