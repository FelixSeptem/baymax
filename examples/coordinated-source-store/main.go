package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
)

func main() {
	dir, err := os.MkdirTemp("", "baymax-coordinated-source-store-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	coordinator, err := sourcecoordination.OpenFile(dir, sourcecoordination.Options{OwnerNamespace: "example/history"})
	if err != nil {
		panic(err)
	}
	defer func() { _ = coordinator.Close() }()
	lease, err := coordinator.Acquire(context.Background(), "example-writer", time.Minute)
	if err != nil {
		panic(err)
	}
	committed, err := coordinator.CompareAndSwap(context.Background(), lease, 0, []byte(`{"source":"example/history","state":"committed"}`))
	if err != nil {
		panic(err)
	}
	fmt.Printf("coordinated-cas=revision:%d\n", committed.Revision)

	local := types.SameHostSingleWriterScope("example/history", "example:local")
	err = local.Check(types.DeploymentRequest{Profile: types.DeploymentProfileMultiInstance})
	fmt.Printf("local-multi-instance-rejected=%t\n", err != nil)
	fmt.Printf("cross-machine-supported=%t\n", coordinator.DeploymentScope().Supports(types.DeploymentRequest{Profile: types.DeploymentProfileCrossMachine}))
}
