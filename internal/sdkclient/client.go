package sdkclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Client is a minimal Arcane API client using API key auth.
type Client struct {
	BaseURL *url.URL
	APIKey  string
	http    *http.Client
}

func NewClient(endpoint, apiKey string) *Client {
	return NewClientWithTimeout(endpoint, apiKey, 30*time.Second)
}

func NewClientWithTimeout(endpoint, apiKey string, timeout time.Duration) *Client {
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	u, _ := url.Parse(endpoint)
	return &Client{
		BaseURL: u,
		APIKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) newRequest(ctx context.Context, method, p string, body any) (*http.Request, error) {
	rel := &url.URL{Path: path.Join(c.BaseURL.Path, p)}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)
	return req, nil
}

func (c *Client) do(req *http.Request, v any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return fmt.Errorf("arcane API error: %s: %s", res.Status, strings.TrimSpace(string(b)))
	}
	if v == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}
	dec := json.NewDecoder(res.Body)
	return dec.Decode(v)
}

// Models derived from api-1.json
// components/schemas/UserCreateUser
type CreateUserRequest struct {
	DisplayName *string  `json:"displayName,omitempty"`
	Email       *string  `json:"email,omitempty"`
	Locale      *string  `json:"locale,omitempty"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles,omitempty"`
	Username    string   `json:"username"`
}

// components/schemas/UserUpdateUser
type UpdateUserRequest struct {
	DisplayName *string  `json:"displayName,omitempty"`
	Email       *string  `json:"email,omitempty"`
	Locale      *string  `json:"locale,omitempty"`
	Password    *string  `json:"password,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// components/schemas/UserUser
type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Display   *string  `json:"displayName,omitempty"`
	Email     *string  `json:"email,omitempty"`
	Locale    *string  `json:"locale,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	CreatedAt *string  `json:"createdAt,omitempty"`
	UpdatedAt *string  `json:"updatedAt,omitempty"`
}

// components/schemas/BaseApiResponseUser
type userResponse struct {
	Success bool `json:"success"`
	Data    User `json:"data"`
}

// CreateUser POST /users
func (c *Client) CreateUser(ctx context.Context, reqBody CreateUserRequest) (*User, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "users", reqBody)
	if err != nil {
		return nil, err
	}
	var out userResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// GetUser GET /users/{id}
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("users", id), nil)
	if err != nil {
		return nil, err
	}
	var out userResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// UpdateUser PUT /users/{id}
func (c *Client) UpdateUser(ctx context.Context, id string, body UpdateUserRequest) (*User, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("users", id), body)
	if err != nil {
		return nil, err
	}
	var out userResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// DeleteUser DELETE /users/{id}
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("users", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- Settings --------
// components/schemas/SettingsPublicSetting
type SettingsPublicSetting struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// UpdateSettings PUT /environments/{id}/settings
func (c *Client) UpdateSettings(ctx context.Context, envID string, values map[string]string) ([]SettingsPublicSetting, error) {
	// Send raw map[string]string matching SettingsUpdate fields
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("environments", envID, "settings"), values)
	if err != nil {
		return nil, err
	}
	// Response: BaseApiResponseListSettingDto -> data: []SettingsSettingDto or public
	var out struct {
		Success bool                    `json:"success"`
		Data    []SettingsPublicSetting `json:"data"`
	}
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetSettings GET /environments/{id}/settings
func (c *Client) GetSettings(ctx context.Context, envID string) (map[string]string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", envID, "settings"), nil)
	if err != nil {
		return nil, err
	}
	var arr []SettingsPublicSetting
	if err := c.do(req, &arr); err != nil {
		return nil, err
	}
	res := make(map[string]string, len(arr))
	for _, s := range arr {
		res[s.Key] = s.Value
	}
	return res, nil
}

// -------- Projects --------
type ProjectCreateRequest struct {
	ComposeContent string  `json:"composeContent"`
	EnvContent     *string `json:"envContent,omitempty"`
	Name           string  `json:"name"`
}

type ProjectCreateResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	ServiceCount int    `json:"serviceCount"`
	RunningCount int    `json:"runningCount"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type projectCreateEnvelope struct {
	Success bool                  `json:"success"`
	Data    ProjectCreateResponse `json:"data"`
}

type ProjectUpdateRequest struct {
	ComposeContent *string `json:"composeContent,omitempty"`
	EnvContent     *string `json:"envContent,omitempty"`
	Name           *string `json:"name,omitempty"`
}

type ProjectDetails struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	ServiceCount   int     `json:"serviceCount"`
	RunningCount   int     `json:"runningCount"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
	ComposeContent *string `json:"composeContent,omitempty"`
	EnvContent     *string `json:"envContent,omitempty"`
}

type projectDetailsEnvelope struct {
	Success bool           `json:"success"`
	Data    ProjectDetails `json:"data"`
}

type ProjectDestroyOptions struct {
	RemoveFiles   bool `json:"removeFiles"`
	RemoveVolumes bool `json:"removeVolumes"`
}

func (c *Client) CreateProject(ctx context.Context, envID string, body ProjectCreateRequest) (*ProjectCreateResponse, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects"), body)
	if err != nil {
		return nil, err
	}
	var env projectCreateEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetProject(ctx context.Context, envID, projectID string) (*ProjectDetails, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", envID, "projects", projectID), nil)
	if err != nil {
		return nil, err
	}
	var env projectDetailsEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateProject(ctx context.Context, envID, projectID string, body ProjectUpdateRequest) (*ProjectDetails, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("environments", envID, "projects", projectID), body)
	if err != nil {
		return nil, err
	}
	var env projectDetailsEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeployProject POST /environments/{id}/projects/{projectId}/up
func (c *Client) DeployProject(ctx context.Context, envID, projectID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects", projectID, "up"), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) DestroyProject(ctx context.Context, envID, projectID string, opts ProjectDestroyOptions) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("environments", envID, "projects", projectID, "destroy"), opts)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- Notifications --------
type NotificationUpdate struct {
	Provider string         `json:"provider"`
	Enabled  bool           `json:"enabled"`
	Config   map[string]any `json:"config"`
}

type NotificationResponse struct {
	ID       int64          `json:"id"`
	Provider string         `json:"provider"`
	Enabled  bool           `json:"enabled"`
	Config   map[string]any `json:"config"`
}

func (c *Client) UpsertNotification(ctx context.Context, envID string, body NotificationUpdate) (*NotificationResponse, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "notifications", "settings"), body)
	if err != nil {
		return nil, err
	}
	var out NotificationResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetNotification(ctx context.Context, envID, provider string) (*NotificationResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", envID, "notifications", "settings", provider), nil)
	if err != nil {
		return nil, err
	}
	var out NotificationResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteNotification(ctx context.Context, envID, provider string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("environments", envID, "notifications", "settings", provider), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- Containers --------
type ContainerCreateRequest struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	AutoRemove      *bool             `json:"autoRemove,omitempty"`
	Command         []string          `json:"command,omitempty"`
	CPUs            *float64          `json:"cpus,omitempty"`
	Entrypoint      []string          `json:"entrypoint,omitempty"`
	Environment     []string          `json:"environment,omitempty"`
	Memory          *int64            `json:"memory,omitempty"`
	Networks        []string          `json:"networks,omitempty"`
	Ports           map[string]string `json:"ports,omitempty"`
	Privileged      *bool             `json:"privileged,omitempty"`
	RestartPolicy   *string           `json:"restartPolicy,omitempty"`
	User            *string           `json:"user,omitempty"`
	Volumes         []string          `json:"volumes,omitempty"`
	WorkingDir      *string           `json:"workingDir,omitempty"`
	Hostname        *string           `json:"hostname,omitempty"`
	Domainname      *string           `json:"domainname,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	TTY             *bool             `json:"tty,omitempty"`
	AttachStdin     *bool             `json:"attachStdin,omitempty"`
	AttachStdout    *bool             `json:"attachStdout,omitempty"`
	AttachStderr    *bool             `json:"attachStderr,omitempty"`
	OpenStdin       *bool             `json:"openStdin,omitempty"`
	StdinOnce       *bool             `json:"stdinOnce,omitempty"`
	NetworkDisabled *bool             `json:"networkDisabled,omitempty"`
}

type ContainerCreated struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

type containerCreatedEnvelope struct {
	Success bool             `json:"success"`
	Data    ContainerCreated `json:"data"`
}

type ContainerDetails struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Created string `json:"created"`
	Status  string `json:"status"`
}

type containerDetailsEnvelope struct {
	Success bool             `json:"success"`
	Data    ContainerDetails `json:"data"`
}

func (c *Client) CreateContainer(ctx context.Context, envID string, body ContainerCreateRequest) (*ContainerCreated, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "containers"), body)
	if err != nil {
		return nil, err
	}
	var env containerCreatedEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetContainer(ctx context.Context, envID, containerID string) (*ContainerDetails, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", envID, "containers", containerID), nil)
	if err != nil {
		return nil, err
	}
	var env containerDetailsEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteContainer(ctx context.Context, envID, containerID string, force, volumes bool) error {
	// These are query parameters per OpenAPI
	p := path.Join("environments", envID, "containers", containerID)
	u := *c.BaseURL
	u.Path = path.Join(c.BaseURL.Path, p)
	q := u.Query()
	if force {
		q.Set("force", "true")
	}
	if volumes {
		q.Set("volumes", "true")
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)
	return c.do(req, nil)
}

// -------- Container Registries --------
type CreateContainerRegistryRequest struct {
	URL         string  `json:"url"`
	Username    string  `json:"username"`
	Token       string  `json:"token"`
	Description *string `json:"description,omitempty"`
	Insecure    *bool   `json:"insecure,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

type UpdateContainerRegistryRequest struct {
	URL         *string `json:"url,omitempty"`
	Username    *string `json:"username,omitempty"`
	Token       *string `json:"token,omitempty"`
	Description *string `json:"description,omitempty"`
	Insecure    *bool   `json:"insecure,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

type ContainerRegistry struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Description string `json:"description"`
	Insecure    bool   `json:"insecure"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type containerRegistryEnvelope struct {
	Success bool              `json:"success"`
	Data    ContainerRegistry `json:"data"`
}

func (c *Client) CreateContainerRegistry(ctx context.Context, body CreateContainerRegistryRequest) (*ContainerRegistry, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "container-registries", body)
	if err != nil {
		return nil, err
	}
	var env containerRegistryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetContainerRegistry(ctx context.Context, id string) (*ContainerRegistry, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("container-registries", id), nil)
	if err != nil {
		return nil, err
	}
	var env containerRegistryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateContainerRegistry(ctx context.Context, id string, body UpdateContainerRegistryRequest) (*ContainerRegistry, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("container-registries", id), body)
	if err != nil {
		return nil, err
	}
	var env containerRegistryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteContainerRegistry(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("container-registries", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- Environments --------
type EnvironmentCreateRequest struct {
	APIURL         string  `json:"apiUrl"`
	Name           *string `json:"name,omitempty"`
	AccessToken    *string `json:"accessToken,omitempty"`
	BootstrapToken *string `json:"bootstrapToken,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	UseAPIKey      *bool   `json:"useApiKey,omitempty"`
}

type EnvironmentUpdateRequest struct {
	APIURL         *string `json:"apiUrl,omitempty"`
	Name           *string `json:"name,omitempty"`
	AccessToken    *string `json:"accessToken,omitempty"`
	BootstrapToken *string `json:"bootstrapToken,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	RegenerateKey  *bool   `json:"regenerateApiKey,omitempty"`
}

type Environment struct {
	ID      string `json:"id"`
	APIURL  string `json:"apiUrl"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"apiKey,omitempty"`
}

type environmentEnvelope struct {
	Success bool        `json:"success"`
	Data    Environment `json:"data"`
}

type EnvironmentAgentPairRequest struct {
	Rotate bool `json:"rotate"`
}

type EnvironmentAgentPairResponse struct {
	Token string `json:"token"`
}

type agentPairEnvelope struct {
	Success bool                         `json:"success"`
	Data    EnvironmentAgentPairResponse `json:"data"`
}

func (c *Client) CreateEnvironment(ctx context.Context, body EnvironmentCreateRequest) (*Environment, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "environments", body)
	if err != nil {
		return nil, err
	}
	var env environmentEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetEnvironment(ctx context.Context, id string) (*Environment, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", id), nil)
	if err != nil {
		return nil, err
	}
	var env environmentEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateEnvironment(ctx context.Context, id string, body EnvironmentUpdateRequest) (*Environment, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("environments", id), body)
	if err != nil {
		return nil, err
	}
	var env environmentEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteEnvironment(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("environments", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) PairEnvironment(ctx context.Context, envID string, rotate bool) (string, error) {
	body := EnvironmentAgentPairRequest{Rotate: rotate}
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "agent", "pair"), body)
	if err != nil {
		return "", err
	}
	var env agentPairEnvelope
	if err := c.do(req, &env); err != nil {
		return "", err
	}
	return env.Data.Token, nil
}

// Project lifecycle: up/down/restart/redeploy
func (c *Client) UpProject(ctx context.Context, envID, projectID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects", projectID, "up"), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) DownProject(ctx context.Context, envID, projectID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects", projectID, "down"), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) RedeployProject(ctx context.Context, envID, projectID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects", projectID, "redeploy"), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// PullProjectImages POST /environments/{id}/projects/{projectId}/pull
func (c *Client) PullProjectImages(ctx context.Context, envID, projectID string) error {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "projects", projectID, "pull"), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- Git Repositories --------
type GitRepositoryCreateRequest struct {
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	AuthType    string  `json:"authType"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	SSHKey      *string `json:"sshKey,omitempty"`
	Token       *string `json:"token,omitempty"`
	Username    *string `json:"username,omitempty"`
}

type GitRepositoryUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	AuthType    *string `json:"authType,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	SSHKey      *string `json:"sshKey,omitempty"`
	Token       *string `json:"token,omitempty"`
	Username    *string `json:"username,omitempty"`
}

type GitRepository struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	AuthType    string `json:"authType"`
	Enabled     bool   `json:"enabled"`
	Username    string `json:"username"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type gitRepositoryEnvelope struct {
	Success bool          `json:"success"`
	Data    GitRepository `json:"data"`
}

func (c *Client) CreateGitRepository(ctx context.Context, body GitRepositoryCreateRequest) (*GitRepository, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "customize/git-repositories", body)
	if err != nil {
		return nil, err
	}
	var env gitRepositoryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetGitRepository(ctx context.Context, id string) (*GitRepository, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("customize/git-repositories", id), nil)
	if err != nil {
		return nil, err
	}
	var env gitRepositoryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateGitRepository(ctx context.Context, id string, body GitRepositoryUpdateRequest) (*GitRepository, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("customize/git-repositories", id), body)
	if err != nil {
		return nil, err
	}
	var env gitRepositoryEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteGitRepository(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("customize/git-repositories", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- API Keys --------
type CreateApiKeyRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	ExpiresAt   *string `json:"expiresAt,omitempty"` // RFC3339 date-time
}

type UpdateApiKeyRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	ExpiresAt   *string `json:"expiresAt,omitempty"` // RFC3339 date-time
}

type ApiKey struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	KeyPrefix   string  `json:"keyPrefix"`
	UserID      string  `json:"userId"`
	ExpiresAt   *string `json:"expiresAt,omitempty"`
	LastUsedAt  *string `json:"lastUsedAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt,omitempty"`
}

type ApiKeyCreated struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Key         string  `json:"key"` // Only returned on creation
	KeyPrefix   string  `json:"keyPrefix"`
	UserID      string  `json:"userId"`
	ExpiresAt   *string `json:"expiresAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   *string `json:"updatedAt,omitempty"`
}

type apiKeyEnvelope struct {
	Success bool   `json:"success"`
	Data    ApiKey `json:"data"`
}

type apiKeyCreatedEnvelope struct {
	Success bool          `json:"success"`
	Data    ApiKeyCreated `json:"data"`
}

// CreateApiKey POST /api-keys
func (c *Client) CreateApiKey(ctx context.Context, body CreateApiKeyRequest) (*ApiKeyCreated, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "api-keys", body)
	if err != nil {
		return nil, err
	}
	var env apiKeyCreatedEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// GetApiKey GET /api-keys/{id}
func (c *Client) GetApiKey(ctx context.Context, id string) (*ApiKey, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("api-keys", id), nil)
	if err != nil {
		return nil, err
	}
	var env apiKeyEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateApiKey PUT /api-keys/{id}
func (c *Client) UpdateApiKey(ctx context.Context, id string, body UpdateApiKeyRequest) (*ApiKey, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("api-keys", id), body)
	if err != nil {
		return nil, err
	}
	var env apiKeyEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteApiKey DELETE /api-keys/{id}
func (c *Client) DeleteApiKey(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("api-keys", id), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// -------- GitOps Syncs --------
type GitOpsSyncCreateRequest struct {
	Name         string  `json:"name"`
	RepositoryID string  `json:"repositoryId"`
	Branch       string  `json:"branch"`
	ComposePath  string  `json:"composePath"`
	ProjectName  *string `json:"projectName,omitempty"`
	AutoSync     *bool   `json:"autoSync,omitempty"`
	SyncInterval *int64  `json:"syncInterval,omitempty"`
	// Note: 'enabled' is read-only and not included in create requests
}

type GitOpsSyncUpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	RepositoryID *string `json:"repositoryId,omitempty"`
	Branch       *string `json:"branch,omitempty"`
	ComposePath  *string `json:"composePath,omitempty"`
	ProjectName  *string `json:"projectName,omitempty"`
	AutoSync     *bool   `json:"autoSync,omitempty"`
	SyncInterval *int64  `json:"syncInterval,omitempty"`
	// Note: 'enabled' is read-only and not included in update requests
}

type GitOpsSync struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	EnvironmentID  string  `json:"environmentId"`
	RepositoryID   string  `json:"repositoryId"`
	Branch         string  `json:"branch"`
	ComposePath    string  `json:"composePath"`
	ProjectName    string  `json:"projectName"`
	ProjectID      *string `json:"projectId,omitempty"`
	AutoSync       bool    `json:"autoSync"`
	SyncInterval   int64   `json:"syncInterval"`
	Enabled        bool    `json:"enabled"`
	LastSyncAt     *string `json:"lastSyncAt,omitempty"`
	LastSyncCommit *string `json:"lastSyncCommit,omitempty"`
	LastSyncStatus *string `json:"lastSyncStatus,omitempty"`
	LastSyncError  *string `json:"lastSyncError,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type gitOpsSyncEnvelope struct {
	Success bool       `json:"success"`
	Data    GitOpsSync `json:"data"`
}

func (c *Client) CreateGitOpsSync(ctx context.Context, envID string, body GitOpsSyncCreateRequest) (*GitOpsSync, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("environments", envID, "gitops-syncs"), body)
	if err != nil {
		return nil, err
	}
	var env gitOpsSyncEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetGitOpsSync(ctx context.Context, envID, syncID string) (*GitOpsSync, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("environments", envID, "gitops-syncs", syncID), nil)
	if err != nil {
		return nil, err
	}
	var env gitOpsSyncEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateGitOpsSync(ctx context.Context, envID, syncID string, body GitOpsSyncUpdateRequest) (*GitOpsSync, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("environments", envID, "gitops-syncs", syncID), body)
	if err != nil {
		return nil, err
	}
	var env gitOpsSyncEnvelope
	if err := c.do(req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteGitOpsSync(ctx context.Context, envID, syncID string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("environments", envID, "gitops-syncs", syncID), nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
