package main

import (
	"github.com/0xalchemist/go-flow-tooling/tooling"
)

func main() {

	flow := tooling.NewFlowConfigLocalhost()

	flow.DeployContract("nft")

}
