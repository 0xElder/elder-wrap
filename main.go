package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/0xElder/elder-wrap/pkg/config"
	"github.com/0xElder/elder-wrap/pkg/elder"
	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/0xElder/elder-wrap/pkg/logging"
	"github.com/0xElder/elder-wrap/pkg/middleware"
	"github.com/0xElder/elder-wrap/pkg/rollapp"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cfg = config.NewConfig()
	loggerOpts := &slog.HandlerOptions{Level: cfg.GetSlogLevel()}
	// Change to JSON/Text logger if needed
	logger := logging.NewDevSlogger(loggerOpts)

	rootCmd := &cobra.Command{
		Use:   "elder-wrap",
		Short: "Elder wrap CLI tool",
	}

	// Create keystore and client
	store, err := keystore.NewPlainKeyStore(cfg.KeyStoreDir)
	if err != nil {
		logger.Error(ctx, "failed to create keystore", "error", err)
		os.Exit(1)
	}
	keystoreClient := keystore.NewKeyStoreClient(store, logger.With("component", "KeyStoreClient"))

	// Add keystore commands
	rootCmd.AddCommand(keystore.GetKeystoreCommands(keystoreClient))

	// Add serve command
	serveCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(ctx, store, logger)
		},
	}
	rootCmd.AddCommand(serveCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Error(ctx, "Elder-wrap failed", "error", err)
		os.Exit(1)
	}
}

// runServer contains the original HTTP server logic
func runServer(ctx context.Context, keystore keystore.KeyStore, logger logging.Logger) error {
	elderClient, err := elder.NewElderClient(cfg.ElderGrpcEndpoint, keystore, logger.With("component", "ElderClient"))
	if err != nil {
		logger.Error(ctx, "failed to create elder client", "error", err)
		return errors.Wrap(err, "failed to create elder client")
	}
	defer elderClient.Conn.Close()

	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return middleware.RestLoggingMiddleware(next, logger)
	})

	rollApps := cfg.ListRollApps()
	for _, rollApp := range rollApps {
		rollAppConfig, err := cfg.GetRollAppConfig(rollApp)
		if err != nil {
			logger.Error(ctx, "failed to get rollapp config", "rollapp", rollApp, "error", err)
			return errors.Wrapf(err, "failed to get rollapp config for %s", rollApp)
		}

		logger.Info(ctx, "Creating rollapp handler", "rollapp", rollApp, "rpc", rollAppConfig.RPC, "elderId", rollAppConfig.ElderRegistrationId)
		rollAppHandler, err := rollapp.NewRollApp(
			rollAppConfig.RPC,
			rollAppConfig.ElderRegistrationId,
			keystore,
			logger.With("rollapp", rollApp),
			elderClient,
		)
		if err != nil {
			logger.Error(ctx, "failed to create rollapp handler", "rollapp", rollApp, "error", err)
			return errors.Wrapf(err, "failed to create rollapp handler for %s", rollApp)
		}

		router.HandleFunc(fmt.Sprintf("/%s", rollApp), rollAppHandler.HandleRequest).Methods(http.MethodPost)
	}

	router.HandleFunc("/", baseHandler).Methods(http.MethodGet)

	logger.Info(ctx, "Starting elder-wrap server", "port", cfg.ElderWrapPort)
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
