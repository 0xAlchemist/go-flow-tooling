package main

import (
	"github.com/0xalchemist/go-flow-tooling/tooling"
)

func main() {

	flow := tooling.NewFlowConfigLocalhost()

	// Deploy Contracts
	flow.DeployContract("nft")
	flow.DeployContract("ft")

	// Send Transaction
	flow.SendTransaction("ft", "create_nft_collection")

	// Run Script
	flow.RunScript("test")
}
