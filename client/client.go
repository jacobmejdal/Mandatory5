package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/jacobmejdal/Mandatory5/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewAuctionClient(conn)

	// Place bid
	bidReply, err := client.Bid(
		context.Background(),
		&pb.BidRequest{
			Bidder: 1,
			Amount: 100,
			Id:     time.Now().UnixNano(),
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Bid outcome:", bidReply.Outcome)

	// Query auction status
	result, err := client.Result(
		context.Background(),
		&pb.ResultRequest{},
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Highest bidder: %d\nHighest bid: %d\nAuction over: %t\n",
		result.HighestBidder,
		result.Amount,
		result.AuctionOver,
	)
}