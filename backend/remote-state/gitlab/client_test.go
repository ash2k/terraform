package gitlab

import "github.com/hashicorp/terraform/state/remote"

var (
	_ remote.ClientLocker = &remoteClient{}
)
