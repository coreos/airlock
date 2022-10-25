package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coreos/airlock/internal/lock"
)

var (
	// cmdGetSlots holds `airlock ex get slots`
	cmdGetSlots = &cobra.Command{
		Use:   "slots",
		Short: "Introspect groups/slots state",
		RunE:  runGetSlots,
	}
)

// runGetSlots performs live introspection of groups/slots.
func runGetSlots(cmd *cobra.Command, cmdArgs []string) error {
	if runSettings == nil {
		return errors.New("nil runSettings")
	}

	for group, maxSlots := range runSettings.LockGroups {
		if group == "" {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), runSettings.EtcdTxnTimeout)
		defer cancel()

		manager, err := lock.NewManager(ctx, runSettings.EtcdEndpoints, runSettings.ClientCertPubPath, runSettings.ClientCertKeyPath, runSettings.EtcdTxnTimeout, group, maxSlots)
		if err != nil {
			return err
		}
		semaphore, err := manager.FetchSemaphore(ctx)
		if err != nil {
			return err
		}
		printHumanShort(group, semaphore)
	}

	return nil
}

// printHumanShort prints groups/slots details in a short, human-friendly way.
func printHumanShort(group string, semaphore *lock.Semaphore) {
	if group == "" || semaphore == nil {
		return
	}

	fmt.Printf("group: %s\n", group)
	fmt.Printf(" semaphore slots: %d\n", semaphore.TotalSlots)
	fmt.Printf(" lock owners:\n")
	for _, owner := range semaphore.Holders {
		fmt.Printf(" - %s\n", owner)
	}
	fmt.Printf("\n---\n")
}
