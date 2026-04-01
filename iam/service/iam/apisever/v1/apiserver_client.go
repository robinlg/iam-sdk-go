// Copyright 2025 Robin Liu <robinliu27@163.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package v1

import (
	"github.com/robinlg/iam-sdk-go/rest"
	v1 "github.com/robinlg/iamapi/apiserver/v1"
	"github.com/robinlg/iamlib/pkg/runtime"
)

// APIV1Interface has methods to work with iam resources.
type APIV1Interface interface {
	RESTClient() rest.Interface
	SecretsGetter
	UsersGetter
	PoliciesGetter
}

// APIV1Client is used to interact with features provided by the group.
type APIV1Client struct {
	restClient rest.Interface
}

// Users create and return user rest client.
func (c *APIV1Client) Users() UserInterface {
	return newUsers(c)
}

// Secrets create and return secret rest client.
func (c *APIV1Client) Secrets() SecretInterface {
	return newSecrets(c)
}

// Policies create and return policy rest client.
func (c *APIV1Client) Policies() PolicyInterface {
	return newPolicies(c)
}

// NewForConfig creates a new APIV1Client for the given config.
func NewForConfig(c *rest.Config) (*APIV1Client, error) {
	config := *c
	setConfigDefaults(&config)

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &APIV1Client{client}, nil
}

func setConfigDefaults(config *rest.Config) {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = ""
	config.Negotiator = runtime.NewSimpleClientNegotiator()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultUserAgent()
	}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *APIV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}

	return c.restClient
}
