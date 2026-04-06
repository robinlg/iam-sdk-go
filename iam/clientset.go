package iam

import (
	"github.com/robinlg/iam-sdk-go/iam/service/iam"
	"github.com/robinlg/iam-sdk-go/rest"
)

// Interface defines method used to return client interface used by marmotedu organization.
type Interface interface {
	Iam() iam.IamInterface
	// Tms() tms.TmsInterface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	iam *iam.IamClient
	// tms *tms.TmsClient
}

var _ Interface = &Clientset{}

// Iam retrieves the IamClient.
func (c *Clientset) Iam() iam.IamInterface {
	return c.iam
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c

	var cs Clientset

	var err error

	cs.iam, err = iam.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	/*
		cs.tms, err = tms.NewForConfig(&configShallowCopy)
		if err != nil {
			return nil, err
		}
	*/
	return &cs, nil
}
