package keystore

import (
	"fmt"

	"github.com/spf13/cobra"
)

// GetKeystoreCommands returns keystore commands that can be added to the main elder-wrap CLI
func GetKeystoreCommands(client *KeyStoreClient) *cobra.Command {
	keyStoreCommand := &cobra.Command{
		Use:   "keystore",
		Short: "Manage keys in the keystore",
	}

	// Import key command
	importCmd := &cobra.Command{
		Use:   "import [alias] [private-key-hex]",
		Short: "Import a private key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.ImportPrivateKey(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Imported key with alias: %s\n", args[0])
			return nil
		},
	}

	// List keys command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stored keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := client.ListKeys()
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				fmt.Println("No keys found")
				return nil
			}
			fmt.Println("Stored keys:")
			for alias, key := range keys {
				fmt.Printf("- Alias: %s\n  EVM Address: %s\n  Elder Address: %s\n",
					alias, key.EvmAddress.Hex(), key.ElderAddress)
			}
			return nil
		},
	}

	// Get key command
	getCmd := &cobra.Command{
		Use:   "get [alias]",
		Short: "Get key details by alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := client.GetKeyByAlias(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Key details for alias '%s':\n", args[0])
			fmt.Printf("EVM Address: %s\nElder Address: %s\n",
				key.EvmAddress.Hex(), key.ElderAddress)
			return nil
		},
	}

	// Delete key command
	deleteCmd := &cobra.Command{
		Use:   "delete [alias]",
		Short: "Delete a key by alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.DeleteKey(args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted key with alias: %s\n", args[0])
			return nil
		},
	}

	// Find by EVM address command
	findEvmCmd := &cobra.Command{
		Use:   "find-evm [address]",
		Short: "Find key by EVM address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := client.GetKeyByEvmAddress(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Found key with EVM address '%s':\n", args[0])
			fmt.Printf("Elder Address: %s\n", key.ElderAddress)
			return nil
		},
	}

	// Find by Elder address command
	findElderCmd := &cobra.Command{
		Use:   "find-elder [address]",
		Short: "Find key by Elder address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := client.GetKeyByElderAddress(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Found key with Elder address '%s':\n", args[0])
			fmt.Printf("EVM Address: %s\n", key.EvmAddress.Hex())
			return nil
		},
	}

	keyStoreCommand.AddCommand(
		importCmd,
		listCmd,
		getCmd,
		deleteCmd,
		findEvmCmd,
		findElderCmd,
	)

	return keyStoreCommand
}
