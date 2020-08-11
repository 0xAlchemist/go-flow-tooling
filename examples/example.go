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
	// flow.SendTransaction()

	// Run Script
	// flow.SendScript()
}
