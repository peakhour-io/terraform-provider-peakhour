package client

// Domain types
type Domain struct {
	Name       string       `json:"name"`
	Origins    []OriginAddr `json:"origins,omitempty"`
	SSL        *string      `json:"ssl,omitempty"`
	DNS        *bool        `json:"dns,omitempty"`
	ManageURL  string       `json:"manage_url,omitempty"`
	Authorised []Authorised `json:"authorised,omitempty"`
}

type Authorised struct {
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}

type DomainAdd struct {
	Name string `json:"name"`
}

type Domains struct {
	Domains        []Domain        `json:"domains"`
	GrantedDomains []GrantedDomain `json:"granted_domains"`
}

type GrantedDomain struct {
	Name       string   `json:"name"`
	Owner      string   `json:"owner"`
	Permission []string `json:"permission"`
	ManageURL  string   `json:"manage_url"`
}

// Origin types
type OriginAddr struct {
	Address string `json:"address"`
	Weight  *int   `json:"weight,omitempty"`
}

type OriginPool struct {
	Tag                          string       `json:"tag"`
	Addresses                    []OriginAddr `json:"addresses"`
	ShieldName                   *string      `json:"shield_name,omitempty"`
	LoadBalancingMode            *string      `json:"load_balancing_mode,omitempty"`
	LoadBalancingKey             *string      `json:"load_balancing_key,omitempty"`
	LoadBalancingOverloadPercent *int         `json:"load_balancing_overload_percent,omitempty"`
}

// Service types
type DomainService struct {
	Type string `json:"type"`
}

// ReverseProxy config types
type ReverseProxyConfig struct {
	Websocket          *bool    `json:"websocket,omitempty"`
	Gzip               *bool    `json:"gzip,omitempty"`
	Brotli             *bool    `json:"brotli,omitempty"`
	Aliases            []string `json:"aliases,omitempty"`
	TrackSessions      *bool    `json:"track_sessions,omitempty"`
	Debug              *bool    `json:"debug,omitempty"`
	Segment            *bool    `json:"segment,omitempty"`
	RedirectMode       *string  `json:"redirect_mode,omitempty"`
	RedirectLocation   *string  `json:"redirect_location,omitempty"`
	RedirectStatusCode *int     `json:"redirect_status_code,omitempty"`
}

// Transform settings types
type TransformSettings struct {
	TransformHTML               *bool    `json:"transform_html,omitempty"`
	TransformBeacon             *bool    `json:"transform_beacon,omitempty"`
	TransformLazySizes          *bool    `json:"transform_lazy_sizes,omitempty"`
	TransformMixedContent       *bool    `json:"transform_mixed_content,omitempty"`
	TransformImgDimsToQueryArgs *bool    `json:"transform_img_dims_to_query_args,omitempty"`
	TransformImageQuality       *int     `json:"transform_image_quality,omitempty"`
	TransformImageFormat        *bool    `json:"transform_image_format,omitempty"`
	TransformImageOptimise      *bool    `json:"transform_image_optimise,omitempty"`
	TransformImageAPI           *bool    `json:"transform_image_api,omitempty"`
	TransformHTTPHeaderValue    *string  `json:"transform_http_header_value,omitempty"`
	TransformESI                *bool    `json:"transform_esi,omitempty"`
	TransformRewriteDomains     []string `json:"transform_rewrite_domains,omitempty"`
}

// Rule types
type RulePhase struct {
	UUID      string                      `json:"uuid"`
	Pos       int                         `json:"pos"`
	Enabled   *bool                       `json:"enabled"`
	Name      string                      `json:"name"`
	FilterStr string                      `json:"filter_str"`
	Phase     string                      `json:"phase"`
	Actions   map[string][]map[string]any `json:"actions"`
}

type RulePhaseSummary struct {
	Name      string `json:"name"`
	FilterStr string `json:"filter_str"`
	UUID      string `json:"uuid"`
	Pos       int    `json:"pos"`
	Phase     string `json:"phase"`
	Enabled   *bool  `json:"enabled"`
}

type RulePhaseAdd struct {
	Phase     string                      `json:"phase"`
	Name      string                      `json:"name"`
	FilterStr string                      `json:"filter_str"`
	Actions   map[string][]map[string]any `json:"actions"`
}

type RulePhaseUpdate struct {
	Enabled   *bool                       `json:"enabled,omitempty"`
	Name      *string                     `json:"name,omitempty"`
	FilterStr *string                     `json:"filter_str,omitempty"`
	Actions   map[string][]map[string]any `json:"actions,omitempty"`
}

type UUIDResult struct {
	UUID string `json:"uuid"`
}

// Rate limit types
type RateLimitZone struct {
	Name                      string `json:"name"`
	BlockDurationSec          *int   `json:"block_duration_sec,omitempty"`
	ConcurrentConnections     *int   `json:"concurrent_connections,omitempty"`
	ConnectionsMax            *int   `json:"connections_max,omitempty"`
	ConnectionsIntervalSec    *int   `json:"connections_interval_sec,omitempty"`
	RequestsMax               *int   `json:"requests_max,omitempty"`
	RequestsIntervalSec       *int   `json:"requests_interval_sec,omitempty"`
	ResponseErrorsMax         *int   `json:"response_errors_max,omitempty"`
	ResponseErrorsIntervalSec *int   `json:"response_errors_interval_sec,omitempty"`
}

type RateLimitSettings struct {
	Mode []string `json:"mode"`
}

type RateLimitGlobal struct {
	BlockDurationSec          *int `json:"block_duration_sec,omitempty"`
	ConcurrentConnections     *int `json:"concurrent_connections,omitempty"`
	ConnectionsMax            *int `json:"connections_max,omitempty"`
	ConnectionsIntervalSec    *int `json:"connections_interval_sec,omitempty"`
	RequestsMax               *int `json:"requests_max,omitempty"`
	RequestsIntervalSec       *int `json:"requests_interval_sec,omitempty"`
	ResponseErrorsMax         *int `json:"response_errors_max,omitempty"`
	ResponseErrorsIntervalSec *int `json:"response_errors_interval_sec,omitempty"`
}

type RateLimit struct {
	Global RateLimitGlobal   `json:"global"`
	Config RateLimitSettings `json:"config"`
	Zones  []RateLimitZone   `json:"zones"`
}

// Rule list types
type RuleList struct {
	UUID string   `json:"uuid"`
	Name string   `json:"name"`
	Type string   `json:"type"`
	IPs  []string `json:"ips,omitempty"`
	Strs []string `json:"strs,omitempty"`
	Ints []int    `json:"ints,omitempty"`
}

type RuleListAdd struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	IPs  []string `json:"ips,omitempty"`
	Strs []string `json:"strs,omitempty"`
	Ints []int    `json:"ints,omitempty"`
}

type RuleListSummary struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Image Transform types
type ImageTransformPreset struct {
	ID      int            `json:"id"`
	UUID    string         `json:"uuid"`
	Name    string         `json:"name"`
	Config  map[string]any `json:"config"`
	Created string         `json:"created"`
	Updated *string        `json:"updated,omitempty"`
}

type ImageTransformPresetCreate struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

type ImageTransformPresetUpdate struct {
	Config map[string]any `json:"config"`
}

type ImageTransformPresetList struct {
	Presets []ImageTransformPreset `json:"presets"`
}
