package rest

import (
	"net/url"
	"strings"

	"github.com/robinlg/iam-sdk-go/third_party/forked/gorequest"
	"github.com/robinlg/iamlib/pkg/runtime"
	"github.com/robinlg/iamlib/pkg/scheme"
)

// ClientContentConfig controls how RESTClient communicates with the server.
type ClientContentConfig struct {
	Username string
	Password string

	SecretID  string
	SecretKey string
	// Server requires Bearer authentication. This client will not attempt to use
	// refresh tokens for an OAuth2 flow.
	// TODO: demonstrate an OAuth2 compatible client.
	BearerToken string

	// Path to a file containing a BearerToken.
	// If set, the contents are periodically read.
	// The last successfully read value takes precedence over BearerToken.
	BearerTokenFile string
	TLSClientConfig

	// AcceptContentTypes specifies the types the client will accept and is optional.
	// If not set, ContentType will be used to define the Accept header
	AcceptContentTypes string
	// ContentType specifies the wire format used to communicate with the server.
	// This value will be set as the Accept header on requests made to the server if
	// AcceptContentTypes is not set, and as the default content type on any object
	// sent to the server. If not set, "application/json" is used.
	ContentType  string
	GroupVersion scheme.GroupVersion
	Negotiator   runtime.ClientNegotiator
}

// RESTClient imposes common IAM API conventions on a set of resource paths.
// The baseURL is expected to point to an HTTP or HTTPS path that is the parent
// of one or more resources.  The server should return a decodable API resource
// object, or an api.Status object which contains information about the reason for
// any failure.
//
// Most consumers should use client.New() to get a IAM API client.
type RESTClient struct {
	// base is the root URL for all invocations of the client
	base *url.URL
	// group stand for the client group, eg: iam.api, iam.authz
	group string
	// versionedAPIPath is a path segment connecting the base URL to the resource root
	versionedAPIPath string
	// content describes how a RESTClient encodes and decodes responses.
	content ClientContentConfig
	Client  *gorequest.SuperAgent
}

// NewRESTClient creates a new RESTClient. This client performs generic REST functions
// such as Get, Put, Post, and Delete on specified paths.
func NewRESTClient(baseURL *url.URL, versionedAPIPath string,
	config ClientContentConfig, client *gorequest.SuperAgent) (*RESTClient, error) {
	if len(config.ContentType) == 0 {
		config.ContentType = "application/json"
	}

	base := *baseURL
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	base.RawQuery = ""
	base.Fragment = ""

	return &RESTClient{
		base:             &base,
		group:            config.GroupVersion.Group,
		versionedAPIPath: versionedAPIPath,
		content:          config,
		Client:           client,
	}, nil
}