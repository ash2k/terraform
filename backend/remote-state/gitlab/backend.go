package gitlab

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/xanzy/go-gitlab"
)

const (
	paramBaseUrl              = "base_url"
	paramProjectId            = "project_id"
	paramStateName            = "state_name"
	paramToken                = "token"
	paramSkipCertVerification = "skip_cert_verification"
)

func New() backend.Backend {
	// See https://docs.gitlab.com/ee/user/project/new_ci_build_permissions_model.html#job-token for info
	// about CI_JOB_TOKEN environment variable (used below).

	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			paramBaseUrl: {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GITLAB_BASE_URL", nil),
				Description:  "The GitLab base API URL",
				ValidateFunc: validation.NoZeroValues,
			},
			paramProjectId: {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("CI_PROJECT_ID", nil),
				Description:  "The unique id of a GitLab project",
				ValidateFunc: validation.NoZeroValues,
			},
			paramStateName: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      backend.DefaultStateName,
				Description:  "The name of the state",
				InputDefault: backend.DefaultStateName,
				ValidateFunc: validation.NoZeroValues,
			},
			paramToken: {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"GITLAB_TOKEN", "CI_JOB_TOKEN"}, nil),
				Description:  "The OAuth token used to connect to GitLab",
				Sensitive:    true,
				ValidateFunc: validation.NoZeroValues,
			},
			paramSkipCertVerification: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification",
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend
	client *remoteClient
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	baseURL := data.Get(paramBaseUrl)
	projectId := data.Get(paramProjectId).(string)
	stateName := data.Get(paramStateName).(string)
	token := data.Get(paramToken).(string)

	client := cleanhttp.DefaultPooledClient()
	if data.Get(paramSkipCertVerification).(bool) {
		// ignores TLS verification
		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	gitlabClient := gitlab.NewClient(client, token)
	gitlabClient.UserAgent = "go-gitlab/Terraform"
	if baseURLstr, _ := baseURL.(string); baseURLstr != "" {
		err := gitlabClient.SetBaseURL(baseURLstr)
		if err != nil {
			return err
		}
	}

	b.client = &remoteClient{
		client:    gitlabClient.Terraform,
		projectId: projectId,
		stateName: stateName,
	}
	return nil
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(string) error {
	return backend.ErrWorkspacesNotSupported
}
