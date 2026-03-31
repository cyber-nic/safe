package main

import (
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
)

func main() {
	fmt.Println("safe CLI bootstrap")
	fmt.Println("supported starter items:")

	for _, item := range domain.StarterVaultItems() {
		fmt.Printf("- [%s] %s: %s\n", item.Kind, item.Title, item.Description)
	}

	fmt.Println("canonical starter records:")

	for _, record := range domain.StarterVaultItemRecords() {
		canonical, err := record.CanonicalJSON()
		if err != nil {
			panic(err)
		}

		fmt.Printf("- %s\n", canonical)
	}

	fmt.Println("canonical starter events:")

	for _, record := range domain.StarterVaultEventRecords() {
		canonical, err := record.CanonicalJSON()
		if err != nil {
			panic(err)
		}

		fmt.Printf("- %s\n", canonical)
	}
}
