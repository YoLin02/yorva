package hermes

import (
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/managementhealth"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/skillsmanagement"
)

const apiManagementRequestTimeout = 4 * time.Second

var (
	errAPIManagementUnauthorized = errors.New("Hermes API management authentication failed")
	errAPIManagementResponse     = errors.New("Hermes API management response is invalid")
	errAPIManagementRequest      = errors.New("Hermes API management request failed")
)

type APIManagementReader struct {
	resolve func(string, string) (apiManagementTarget, error)
	fetch   func(context.Context, apiManagementTarget, string, int) ([]byte, error)
	now     func() time.Time
}

func NewAPIManagementReader() *APIManagementReader {
	return &APIManagementReader{
		resolve: resolveAPIManagementTarget,
		fetch:   fetchAPIManagementJSON,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// ResolveInstanceManagement performs no network request and no Runtime
// lifecycle mutation. It publishes the two readers only when exact 0.20.5
// Profile configuration proves a closed loopback target and an unambiguous
// Profile-owned key. A stopped listener remains a query-time stable failure.
func (r *APIManagementReader) ResolveInstanceManagement(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.InstanceManagementFeatures, error) {
	if err := validateAPIManagementInstallation(ctx, installation); err != nil || r == nil || r.resolve == nil {
		return yorvaruntime.InstanceManagementFeatures{}, errAPIManagementUnavailable
	}
	target, err := r.resolve(installation.Version, nativeID)
	if err != nil {
		return yorvaruntime.InstanceManagementFeatures{}, err
	}
	target.clear()
	return yorvaruntime.InstanceManagementFeatures{Health: r, SkillRead: r}, nil
}

func (r *APIManagementReader) InspectRuntimeHealth(context.Context, yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	return yorvaruntime.HealthObservation{}, errAPIManagementUnavailable
}

func (r *APIManagementReader) InspectInstanceHealth(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.HealthObservation, error) {
	if err := validateAPIManagementInstallation(ctx, installation); err != nil {
		return yorvaruntime.HealthObservation{}, err
	}
	target, err := r.target(installation.Version, nativeID)
	if err != nil {
		return yorvaruntime.HealthObservation{}, err
	}
	defer target.clear()
	path, err := target.path("/health/detailed")
	if err != nil {
		return yorvaruntime.HealthObservation{}, errAPIManagementRequest
	}
	body, err := r.fetch(ctx, target, path, managementhealth.MaxDetailedResponseBytes)
	if err != nil {
		return yorvaruntime.HealthObservation{}, err
	}
	detailed, err := managementhealth.ParseDetailedHealth(body)
	if err != nil {
		return yorvaruntime.HealthObservation{}, errAPIManagementResponse
	}
	findings := make([]yorvaruntime.HealthFinding, 0, len(detailed.Checks))
	for _, check := range detailed.Checks {
		state, ok := projectHealthCheckState(check.State)
		if !ok {
			return yorvaruntime.HealthObservation{}, errAPIManagementResponse
		}
		findings = append(findings, yorvaruntime.HealthFinding{Code: check.Name, State: state})
	}
	state := yorvaruntime.HealthUnknown
	switch detailed.State {
	case managementhealth.HealthHealthy:
		state = yorvaruntime.HealthHealthy
	case managementhealth.HealthDegraded:
		state = yorvaruntime.HealthDegraded
	default:
		return yorvaruntime.HealthObservation{}, errAPIManagementResponse
	}
	return yorvaruntime.HealthObservation{State: state, Findings: findings, ObservedAt: r.clock()}, nil
}

func (r *APIManagementReader) ListSkills(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.Skill, error) {
	if err := validateAPIManagementInstallation(ctx, installation); err != nil {
		return nil, err
	}
	target, err := r.target(installation.Version, nativeID)
	if err != nil {
		return nil, err
	}
	defer target.clear()
	path, err := target.path("/v1/skills")
	if err != nil {
		return nil, errAPIManagementRequest
	}
	body, err := r.fetch(ctx, target, path, skillsmanagement.MaxInventoryBodyBytes)
	if err != nil {
		return nil, err
	}
	scope, err := skillsmanagement.NewProfileScope(nativeID)
	if err != nil {
		return nil, errAPIManagementResponse
	}
	inventory, err := skillsmanagement.ParseEnabledInventory(scope, body)
	if err != nil {
		return nil, errAPIManagementResponse
	}
	result := make([]yorvaruntime.Skill, 0, len(inventory.Skills))
	for _, skill := range inventory.Skills {
		result = append(result, skill.RuntimeSkill())
	}
	return result, nil
}

func (r *APIManagementReader) InspectSkill(ctx context.Context, installation yorvaruntime.Installation, nativeID, skillID string) (yorvaruntime.Skill, error) {
	items, err := r.ListSkills(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	for _, item := range items {
		if item.ID == skillID {
			return item, nil
		}
	}
	return yorvaruntime.Skill{}, errAPIManagementResponse
}

func (r *APIManagementReader) target(version, nativeID string) (apiManagementTarget, error) {
	if r == nil || r.resolve == nil || r.fetch == nil {
		return apiManagementTarget{}, errAPIManagementUnavailable
	}
	return r.resolve(version, nativeID)
}

func (r *APIManagementReader) clock() time.Time {
	if r != nil && r.now != nil {
		return r.now()
	}
	return time.Now().UTC()
}

func validateAPIManagementInstallation(ctx context.Context, installation yorvaruntime.Installation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if installation.RuntimeKind != Kind || installation.Version != apiManagementVersion || installation.SupportState != yorvaruntime.DiscoverySupported || installation.Path == "" || !filepath.IsAbs(installation.Path) {
		return errAPIManagementUnavailable
	}
	return nil
}

func projectHealthCheckState(state managementhealth.HealthCheckState) (yorvaruntime.HealthState, bool) {
	switch state {
	case managementhealth.CheckOK:
		return yorvaruntime.HealthHealthy, true
	case managementhealth.CheckDegraded, managementhealth.CheckRetrying:
		return yorvaruntime.HealthDegraded, true
	case managementhealth.CheckUnavailable:
		return yorvaruntime.HealthUnhealthy, true
	default:
		return yorvaruntime.HealthUnknown, false
	}
}

func fetchAPIManagementJSON(ctx context.Context, target apiManagementTarget, path string, maximum int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maximum < 1 || target.host != "127.0.0.1" && target.host != "::1" || target.port < 1 || target.port > 65535 || len(target.key) < 16 || (path != target.pathPrefix+"/health/detailed" && path != target.pathPrefix+"/v1/skills") {
		return nil, errAPIManagementRequest
	}
	address := target.address()
	dialer := &net.Dialer{Timeout: apiManagementRequestTimeout, KeepAlive: -1}
	transport := &http.Transport{
		Proxy:             nil,
		DisableKeepAlives: true,
		DialContext: func(dialCtx context.Context, network, requested string) (net.Conn, error) {
			if requested != address || network != "tcp" && network != "tcp4" && network != "tcp6" {
				return nil, errAPIManagementRequest
			}
			connection, err := dialer.DialContext(dialCtx, network, address)
			if err != nil {
				return nil, err
			}
			host, portText, splitErr := net.SplitHostPort(connection.RemoteAddr().String())
			port, portErr := strconv.Atoi(portText)
			ip := net.ParseIP(strings.Trim(host, "[]"))
			if splitErr != nil || portErr != nil || ip == nil || !ip.IsLoopback() || port != target.port {
				_ = connection.Close()
				return nil, errAPIManagementRequest
			}
			return connection, nil
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   apiManagementRequestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	requestCtx, cancel := context.WithTimeout(ctx, apiManagementRequestTimeout)
	defer cancel()
	requestURL := &url.URL{Scheme: "http", Host: address, Path: path}
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, errAPIManagementRequest
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+string(target.key))
	response, err := client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if requestCtx.Err() != nil {
			return nil, context.DeadlineExceeded
		}
		return nil, errAPIManagementRequest
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, errAPIManagementUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, errAPIManagementRequest
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, errAPIManagementResponse
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(maximum)+1))
	if err != nil {
		return nil, errAPIManagementRequest
	}
	if len(body) == 0 || len(body) > maximum {
		return nil, errAPIManagementResponse
	}
	return body, nil
}

func IsAPIManagementUnavailable(err error) bool { return errors.Is(err, errAPIManagementUnavailable) }
