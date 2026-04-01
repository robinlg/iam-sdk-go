// Copyright 2025 Robin Liu <robinliu27@163.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package iam

import (
	apiv1 "github.com/robinlg/iam-sdk-go/iam/service/iam/apisever/v1"
	authzv1 "github.com/robinlg/iam-sdk-go/iam/service/iam/authz/v1"
	"github.com/robinlg/iam-sdk-go/rest"
)

// IamInterface holds the methods that iam server-supported API services,
// versions and resources.
type IamInterface interface {
	APIV1() apiv1.APIV1Interface
	AuthzV1() authzv1.AuthzV1Interface
}

// IamClient contains the clients for iam service. Each iam service has exactly one
// version included in a IamClient.
type IamClient struct {
	apiV1   *apiv1.APIV1Client
	authzV1 *authzv1.AuthzV1Client
}

// APIV1 retrieves the APIV1Client.
func (c *IamClient) APIV1() apiv1.APIV1Interface {
	return c.apiV1
}

// AuthzV1 retrieves the AuthzV1Client.
func (c *IamClient) AuthzV1() authzv1.AuthzV1Interface {
	return c.authzV1
}

// NewForConfig creates a new IamV1Client for the given config.
func NewForConfig(c *rest.Config) (*IamClient, error) {
	configShallowCopy := *c

	var ic IamClient

	var err error

	ic.apiV1, err = apiv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	ic.authzV1, err = authzv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return &ic, nil
}
