// Copyright 2025 Robin Liu <robinliu27@163.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package v1

// The UserExpansion interface allows manually adding extra methods to the UserInterface.
type UserExpansion interface { // PatchStatus modifies the status of an existing node. It returns the copy
	// of the node that the server returns, or an error.
	// PatchStatus(ctx context.Context, nodeName string, data []byte) (*v1.Node, error)
}
