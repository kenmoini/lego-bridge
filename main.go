package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/platform/config/env"
	"github.com/go-acme/lego/v4/providers/dns/pdns"
	"github.com/go-acme/lego/v4/registration"
)

const (
	envNamespace = "PDNS_"

	EnvAPIKey = envNamespace + "API_KEY"
	EnvAPIURL = envNamespace + "API_URL"

	EnvTTL                = envNamespace + "TTL"
	EnvAPIVersion         = envNamespace + "API_VERSION"
	EnvPropagationTimeout = envNamespace + "PROPAGATION_TIMEOUT"
	EnvPollingInterval    = envNamespace + "POLLING_INTERVAL"
	EnvHTTPTimeout        = envNamespace + "HTTP_TIMEOUT"
	EnvServerName         = envNamespace + "SERVER_NAME"
)

// You'll need a user or account type that implements acme.User
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type APIRequest struct {
	Domains []string `json:"domains"`
}

type APIResponse struct {
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
	Status      string `json:"status"`
}

func RequestCertificates(domainsRequested []string) APIResponse {
	responseJSON := APIResponse{}
	launchedPIDs := []int{}
	pdnsAPIURL, pdnsAPIURLPresent := os.LookupEnv("PDNS_API_URL")
	pdnsAPIKey, pdnsAPIKeyPresent := os.LookupEnv("PDNS_API_KEY")
	acmeServerURL, acmeServerURLPresent := os.LookupEnv("ACME_SERVER_URL")
	emailAddress, emailAddressPresent := os.LookupEnv("EMAIL_ADDRESS")
	dnsServers, dnsServersPresent := os.LookupEnv("DNS_SERVERS")

	if !pdnsAPIURLPresent || !pdnsAPIKeyPresent || !acmeServerURLPresent || !emailAddressPresent {
		log.Fatal("PDNS_API_URL, PDNS_API_VERSION, PDNS_API_KEY, and ACME_SERVER_URL must be set")
		os.Exit(1)
	}

	// Create a user. New accounts need an email and private key to start.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}

	myUser := MyUser{
		Email: emailAddress,
		key:   privateKey,
	}

	config := lego.NewConfig(&myUser)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.
	config.CADirURL = acmeServerURL
	config.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}

	pdnsServerURL, err := url.Parse(pdnsAPIURL)
	if err != nil {
		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}

	pdnsConfig := pdns.Config{
		ServerName:         "localhost", // This is the name of the PowerDNS server
		Host:               pdnsServerURL,
		APIVersion:         0,
		APIKey:             pdnsAPIKey,
		TTL:                100,
		PropagationTimeout: 360 * time.Second,
		PollingInterval:    15 * time.Second,
		HTTPClient: &http.Client{
			Timeout: env.GetOrDefaultSecond(EnvHTTPTimeout, 30*time.Second),
		},
	}
	provider, err := pdns.NewDNSProviderConfig(&pdnsConfig)
	if err != nil {
		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}

	// Setup the Provider
	if !dnsServersPresent {
		err = client.Challenge.SetDNS01Provider(provider)
		if err != nil {
			responseJSON = APIResponse{
				Certificate: "",
				Key:         "",
				Status:      "failed: " + err.Error(),
			}
			log.Fatal(err)
		}
	} else {
		serverArray := strings.Split(dnsServers, ",")
		err = client.Challenge.SetDNS01Provider(provider, dns01.AddRecursiveNameservers(dns01.ParseNameservers(serverArray)))
		if err != nil {
			responseJSON = APIResponse{
				Certificate: "",
				Key:         "",
				Status:      "failed: " + err.Error(),
			}
			log.Fatal(err)
		}
	}
	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}
	myUser.Registration = reg

	// Run background pings to do something weird
	for _, domain := range domainsRequested {
		cmd := exec.Command("./dns-ping.sh", domain)
		cmd.Stdout = os.Stdout
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launchedPIDs = append(launchedPIDs, cmd.Process.Pid)
	}

	request := certificate.ObtainRequest{
		Domains: domainsRequested,
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {

		responseJSON = APIResponse{
			Certificate: "",
			Key:         "",
			Status:      "failed: " + err.Error(),
		}
		log.Fatal(err)
	}

	// for _, pid := range launchedPIDs {
	// 	err := exec.Command("kill", "-9", string(pid)).Run()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	// fmt.Printf("%#v\n", certificates)

	responseJSON = APIResponse{
		Certificate: string(certificates.Certificate),
		Key:         string(certificates.PrivateKey),
		Status:      "success",
	}

	return responseJSON

}

func getCertificate(ctx *gin.Context) {
	var req APIRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := RequestCertificates(req.Domains)
	ctx.JSON(http.StatusOK, resp)
}

func setRouter() *gin.Engine {
	// Disable log's color
	gin.DisableConsoleColor()

	// Creates default gin router with Logger and Recovery middleware already attached
	router := gin.New()

	// Global middleware
	// Logger middleware will write the logs to gin.DefaultWriter even if you set with GIN_MODE=release.
	// By default gin.DefaultWriter = os.Stdout
	router.Use(gin.Logger())

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())

	// same as
	// config := cors.DefaultConfig()
	// config.AllowAllOrigins = true
	// router.Use(cors.New(config))
	//router.Use(cors.Default())
	config := cors.DefaultConfig()
	config.AddAllowHeaders("Authorization")
	config.AddAllowHeaders("Content-Type")
	config.AllowCredentials = true
	config.AllowAllOrigins = true
	//config.AllowOriginFunc = func(origin string) bool {
	//		return true
	//}
	router.Use(cors.New(config))
	//router.Use(CORSMiddleware())

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	router.RedirectTrailingSlash = true

	// If enabled, the router checks if another method is allowed for the
	// current route. If this is the case, the router writes a response with
	// the status code 405, "Method Not Allowed". If there is no other method
	// available to handle the request, the router returns the status code 404,
	// "Not Found".
	router.HandleMethodNotAllowed = true

	rootAPI := router.Group("/")
	{
		// Add /hello GET route to router and define route handler function
		rootAPI.GET("/healthz", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"status": "ok"})
		})
		rootAPI.POST("/get-certificate", getCertificate)
	}

	// 404 and 405 handlers
	router.NoMethod(func(ctx *gin.Context) { ctx.JSON(http.StatusMethodNotAllowed, gin.H{}) })
	router.NoRoute(func(ctx *gin.Context) { ctx.JSON(http.StatusNotFound, gin.H{}) })

	return router
}

func main() {
	serverPort, serverPortPresent := os.LookupEnv("SERVER_PORT")
	serverAddress, serverAddressPresent := os.LookupEnv("SERVER_ADDRESS")
	if !serverPortPresent {
		serverPort = "8080"
	}
	if !serverAddressPresent {
		serverAddress = "0.0.0.0"
	}

	Router := setRouter()

	Router.Run(serverAddress + ":" + serverPort)
}
