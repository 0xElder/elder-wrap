package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/0xElder/elder-wrap/pkg/config"
	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/0xElder/elder-wrap/pkg/middleware"
	"github.com/0xElder/elder-wrap/pkg/rollapp"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var cfg *config.Config

func main() {
	cfg = config.NewConfig()

	rootCmd := &cobra.Command{
		Use:   "elder-wrap",
		Short: "Elder wrap CLI tool",
	}

	// Create keystore and client
	store, err := keystore.NewPlainKeyStore(cfg.KeyStoreDir)
	if err != nil {
		log.Fatalf("Failed to create keystore: %v\n", err)
	}
	keystoreClient := keystore.NewKeyStoreClient(store)

	// Add keystore commands
	rootCmd.AddCommand(keystore.GetKeystoreCommands(keystoreClient))

	// Add serve command
	serveCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(store)
		},
	}
	rootCmd.AddCommand(serveCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runServer contains the original HTTP server logic
func runServer(keystore *keystore.PlainKeyStore) error {
	elderConn, err := grpc.NewClient(cfg.ElderGrpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to elder: %v", err)
	}
	defer elderConn.Close()

	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware)

	rollApps := cfg.ListRollApps()
	for _, rollApp := range rollApps {
		rollAppConfig, err := cfg.GetRollAppConfig(rollApp)
		if err != nil {
			return fmt.Errorf("failed to get rollapp config for %s: %v", rollApp, err)
		}

		rollAppHandler, err := rollapp.NewRollApp(
			rollAppConfig.RPC,
			rollAppConfig.ElderRegistrationId,
			keystore,
			elderConn,
		)
		if err != nil {
			return fmt.Errorf("failed to create rollapp handler for %s: %v", rollApp, err)
		}

		router.HandleFunc(fmt.Sprintf("/%s", rollApp), rollAppHandler.HandleRequest).Methods(http.MethodPost)
	}

	router.HandleFunc("/", baseHandler).Methods(http.MethodGet)

	fmt.Printf("Starting server on port %s\n", cfg.ElderWrapPort)
	addr := net.JoinHostPort("", cfg.ElderWrapPort)
	return http.ListenAndServe(addr, router)
}

func baseHandler(w http.ResponseWriter, r *http.Request) {
	endpoints := make(map[string]interface{})
	rollApps := cfg.ListRollApps()

	for _, rollApp := range rollApps {
		rollAppConfig, err := cfg.GetRollAppConfig(rollApp)
		if err != nil {
			continue
		}
		endpoints[rollApp] = map[string]interface{}{
			"endpoint":              fmt.Sprintf("/%s", rollApp),
			"rpc":                   rollAppConfig.RPC,
			"elder_registration_id": rollAppConfig.ElderRegistrationId,
		}
	}

	response := map[string]interface{}{
		"elder_grpc": cfg.ElderGrpcEndpoint,
		"endpoints":  endpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
