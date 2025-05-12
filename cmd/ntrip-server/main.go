package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/admin"
	"github.com/go-gnss/ntrip/auth"
	"github.com/go-gnss/ntrip/internal/inmemory"
	"github.com/go-gnss/ntrip/rtsp"
	"github.com/go-gnss/ntrip/v1source"
	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	httpPort := flag.Int("http-port", 2101, "HTTP port for NTRIP v1/v2")
	rtspPort := flag.Int("rtsp-port", 554, "RTSP port for NTRIP over RTSP")
	v1SourcePort := flag.Int("v1source-port", 2102, "Port for NTRIP v1 SOURCE requests")
	adminPort := flag.Int("admin-port", 8080, "Port for admin API")
	dbPath := flag.String("db-path", "data/ntrip.db", "Path to SQLite database file")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file for admin API")
	tlsKey := flag.String("tls-key", "", "Path to TLS certificate key file for admin API")
	flag.Parse()

	// Set up logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatalf("Invalid log level: %s", *logLevel)
	}
	logger.SetLevel(level)

	// Create authentication manager
	authManager := auth.NewAuthManager()

	// Create no auth for public mounts
	noAuth := auth.NewNoAuth()

	// Create the admin API server first to get access to the database
	var adminServer *admin.Server
	var err error

	// Create the admin server with TLS if certificates are provided
	if *tlsCert != "" && *tlsKey != "" {
		adminServer, err = admin.NewTLSServer(fmt.Sprintf(":%d", *adminPort), *dbPath, *tlsCert, *tlsKey, logger)
	} else {
		adminServer, err = admin.NewServer(fmt.Sprintf(":%d", *adminPort), *dbPath, logger)
	}

	if err != nil {
		logger.Fatalf("Failed to create admin server: %v", err)
	}

	// Create database-backed authenticator
	dbAuth := auth.NewDBAuth(adminServer.GetDB())

	// Set up authentication for mountpoints
	// Get all mountpoints from the database
	mountpoints, err := adminServer.GetDB().ListMountpoints()
	if err != nil {
		logger.Warnf("Failed to list mountpoints from database: %v", err)
		// Continue with empty list if there's an error
		mountpoints = []admin.Mountpoint{}
	}

	// Set authenticator for each mountpoint
	for _, mount := range mountpoints {
		if mount.Status == "online" {
			authManager.SetMountAuthenticator(mount.Name, dbAuth)
			logger.Infof("Set up authentication for mountpoint: %s", mount.Name)
		}
	}

	// Set default authenticator
	authManager.SetDefaultAuthenticator(dbAuth)

	// Set up public mounts
	authManager.SetMountAuthenticator("PUBLIC1", noAuth)

	// Create an authorizer that uses the auth manager
	authorizer := &AuthManagerAuthorizer{
		AuthManager: authManager,
	}

	// Create the source service
	svc := inmemory.NewSourceService(authorizer)

	// Add some example mounts to the sourcetable
	svc.Sourcetable = ntrip.Sourcetable{
		Casters: []ntrip.CasterEntry{
			{
				Host:                "localhost",
				Port:                *httpPort,
				Identifier:          "NTRIP Caster",
				Operator:            "Example Operator",
				NMEA:                true,
				Country:             "DEU",
				Latitude:            50.0,
				Longitude:           8.0,
				FallbackHostAddress: "0.0.0.0",
				FallbackHostPort:    0,
			},
		},
		Networks: []ntrip.NetworkEntry{
			{
				Identifier:          "EXAMPLE",
				Operator:            "Example Operator",
				Authentication:      "B",
				Fee:                 false,
				NetworkInfoURL:      "http://example.com",
				StreamInfoURL:       "http://example.com/streams",
				RegistrationAddress: "register@example.com",
			},
		},
		Mounts: []ntrip.StreamEntry{
			{
				Name:           "SECURE1",
				Identifier:     "SECURE1",
				Format:         "RTCM 3.2",
				FormatDetails:  "1004(1),1005(5),1006(5),1008(5),1012(1),1013(5),1033(5)",
				Carrier:        "2",
				NavSystem:      "GPS+GLO",
				Network:        "EXAMPLE",
				CountryCode:    "DEU",
				Latitude:       50.0,
				Longitude:      8.0,
				NMEA:           false,
				Solution:       false,
				Generator:      "GNSS Receiver",
				Compression:    "none",
				Authentication: "B",
				Fee:            false,
				Bitrate:        9600,
			},
			{
				Name:           "SECURE2",
				Identifier:     "SECURE2",
				Format:         "RTCM 3.3",
				FormatDetails:  "1004(1),1005(5),1006(5),1008(5),1012(1),1013(5),1033(5)",
				Carrier:        "2",
				NavSystem:      "GPS+GLO+GAL",
				Network:        "EXAMPLE",
				CountryCode:    "DEU",
				Latitude:       50.1,
				Longitude:      8.1,
				NMEA:           true,
				Solution:       true,
				Generator:      "GNSS Receiver",
				Compression:    "none",
				Authentication: "D",
				Fee:            false,
				Bitrate:        9600,
			},
			{
				Name:           "PUBLIC1",
				Identifier:     "PUBLIC1",
				Format:         "RTCM 3.2",
				FormatDetails:  "1004(1),1005(5),1006(5),1008(5),1012(1),1013(5),1033(5)",
				Carrier:        "2",
				NavSystem:      "GPS+GLO",
				Network:        "EXAMPLE",
				CountryCode:    "DEU",
				Latitude:       50.2,
				Longitude:      8.2,
				NMEA:           false,
				Solution:       false,
				Generator:      "GNSS Receiver",
				Compression:    "none",
				Authentication: "N",
				Fee:            false,
				Bitrate:        9600,
			},
		},
	}

	// Create a cancellable context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the HTTP caster
	caster := ntrip.NewCaster(fmt.Sprintf(":%d", *httpPort), svc, logger)

	// Create the RTSP server
	rtspHandler := rtsp.RTSPHandler(svc, logger)
	rtspServer := rtsp.NewServer(fmt.Sprintf(":%d", *rtspPort), rtspHandler, logger)

	// Create the v1 SOURCE server
	v1SourceServer := v1source.NewServer(fmt.Sprintf(":%d", *v1SourcePort), svc, logger)

	// Admin API server already created above

	// Start the servers in goroutines
	go func() {
		logger.Infof("Starting HTTP caster on port %d", *httpPort)
		if err := caster.ListenAndServe(); err != nil {
			logger.Fatalf("HTTP caster error: %v", err)
		}
	}()

	go func() {
		logger.Infof("Starting RTSP server on port %d", *rtspPort)
		if err := rtspServer.ListenAndServe(); err != nil {
			logger.Fatalf("RTSP server error: %v", err)
		}
	}()

	go func() {
		logger.Infof("Starting v1 SOURCE server on port %d", *v1SourcePort)
		if err := v1SourceServer.ListenAndServe(); err != nil {
			logger.Fatalf("v1 SOURCE server error: %v", err)
		}
	}()

	go func() {
		if *tlsCert != "" && *tlsKey != "" {
			logger.Infof("Starting admin API server with TLS on port %d", *adminPort)
			if err := adminServer.ListenAndServeTLS(*tlsCert, *tlsKey); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("Admin API server error: %v", err)
			}
		} else {
			logger.Infof("Starting admin API server on port %d", *adminPort)
			if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("Admin API server error: %v", err)
			}
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Shutdown gracefully
	logger.Info("Shutting down servers...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close the servers
	if err := caster.Shutdown(ctx); err != nil {
		logger.Errorf("Error closing HTTP caster: %v", err)
	}

	if err := rtspServer.Close(); err != nil {
		logger.Errorf("Error closing RTSP server: %v", err)
	}

	if err := v1SourceServer.Close(); err != nil {
		logger.Errorf("Error closing v1 SOURCE server: %v", err)
	}

	if err := adminServer.Shutdown(ctx); err != nil {
		logger.Errorf("Error closing admin API server: %v", err)
	}

	// Close the admin database
	if err := adminServer.Close(); err != nil {
		logger.Errorf("Error closing admin database: %v", err)
	}

	logger.Info("All servers shut down")
}

// AuthManagerAuthorizer implements the inmemory.Authoriser interface using the auth.AuthManager
type AuthManagerAuthorizer struct {
	AuthManager *auth.AuthManager
}

// Authorise implements the inmemory.Authoriser interface
func (a *AuthManagerAuthorizer) Authorise(action inmemory.Action, mount, username, password string) (bool, error) {
	// For publish actions, authenticate the mountpoint
	if action == inmemory.PublishAction {
		// Use the database authenticator to validate mountpoint credentials
		if dbAuth, ok := a.AuthManager.GetAuthenticator(mount).(*auth.DBAuth); ok {
			return dbAuth.AuthenticateMountpoint(mount, password)
		}
	}

	// For subscribe actions, authenticate the user
	// Create a mock request with basic auth
	req, err := http.NewRequest("GET", "http://example.com/"+mount, nil)
	if err != nil {
		return false, err
	}

	// Set basic auth if credentials are provided
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Get the authenticator for this mount
	authenticator := a.AuthManager.GetAuthenticator(mount)

	// Check if the mount requires authentication
	if authenticator.Method() == auth.None {
		return true, nil
	}

	// Authenticate the request
	return authenticator.Authenticate(req, mount)
}
