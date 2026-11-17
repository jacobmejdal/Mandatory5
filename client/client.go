package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"
    "strconv"

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

	if len(os.Args) < 3 {
    log.Fatal("usage: go run client.go <bidder> <amount>")
	}

	bidder, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	amount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
    "Submitting bid: bidder=%d amount=%d\n",
    bidder,
    amount,
	)

	// Place bid
	bidReply, err := client.Bid(
		context.Background(),
		&pb.BidRequest{
			Bidder: int32(bidder),
			Amount: int32(amount),
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