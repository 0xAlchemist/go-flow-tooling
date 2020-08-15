package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/enescakir/emoji"
	"github.com/olekukonko/tablewriter"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

func main() {

	//this proabaly does not work yet
	devPointer := flag.Bool("dev", false, "Set to enable devnet")

	flag.Parse()
	host := "127.0.0.1:3569"
	if *devPointer {
		host = "access-001.devnet7.nodes.onflow.org:9000"
	}

	c, err := client.New(host, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Could create a new Flow client", emoji.PileOfPoo)
	}

	args := flag.Args()

	if len(args) != 1 {
		log.Fatalf("%v This command takes a single argument that is an flow account address without 0x prefix", emoji.PileOfPoo)
	}

	ctx := context.Background()

	signerAccount := flag.Args()[0]
	account, err := c.GetAccount(ctx, flow.HexToAddress(signerAccount))
	if err != nil {
		log.Fatalf("%v Could not get public account object for address: %s", emoji.PileOfPoo, signerAccount)
	}

	fmt.Printf("%v  Account: %s \n%v Balance: %d\n", emoji.Label, account.Address, emoji.MoneyBag, account.Balance)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Weight", "SigAlgo", "HashAlgo", "PublicKey"})

	for i, key := range account.Keys {
		table.Append([]string{fmt.Sprintf("%v %d", emoji.Key, i), fmt.Sprintf("%d", key.Weight), fmt.Sprintf("%v", key.SigAlgo), fmt.Sprintf("%v", key.HashAlgo), fmt.Sprintf("%v", key.PublicKey)})
	}
	table.Render()

	if len(account.Code) > 0 {
		fmt.Printf("%v Code:%s", emoji.Clipboard, string(account.Code))
	} else {
		fmt.Printf("No code deployed")
	}
}
