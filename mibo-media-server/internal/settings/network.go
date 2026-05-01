package settings

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const networkCategory = "network"

const (
	networkLocalNetworksKey               = "local_networks"
	networkLocalIPAddressKey              = "local_ip_address"
	networkLocalHTTPPortKey               = "local_http_port"
	networkLocalHTTPSPortKey              = "local_https_port"
	networkAllowRemoteAccessKey           = "allow_remote_access"
	networkRemoteIPFilterKey              = "remote_ip_filter"
	networkRemoteIPFilterModeKey          = "remote_ip_filter_mode"
	networkPublicHTTPPortKey              = "public_http_port"
	networkPublicHTTPSPortKey             = "public_https_port"
	networkExternalDomainKey              = "external_domain"
	networkTrustProxyHeadersKey           = "trust_proxy_headers"
	networkSSLCertificatePathKey          = "ssl_certificate_path"
	networkCertificatePasswordKey         = "certificate_password"
	networkSecureConnectionModeKey        = "secure_connection_mode"
	networkAutomaticPortMappingKey        = "automatic_port_mapping"
	networkMaxVideoStreamsKey             = "max_video_streams"
	networkRemoteStreamingBitrateLimitKey = "remote_streaming_bitrate_limit"
	networkRequestProtocolKey             = "network_request_protocol"
)

type NetworkSettings struct {
	LocalNetworks               []string              `json:"local_networks"`
	LocalIPAddress              string                `json:"local_ip_address"`
	LocalHTTPPort               int                   `json:"local_http_port"`
	LocalHTTPSPort              int                   `json:"local_https_port"`
	AllowRemoteAccess           bool                  `json:"allow_remote_access"`
	RemoteIPFilter              []string              `json:"remote_ip_filter"`
	RemoteIPFilterMode          string                `json:"remote_ip_filter_mode"`
	PublicHTTPPort              int                   `json:"public_http_port"`
	PublicHTTPSPort             int                   `json:"public_https_port"`
	ExternalDomain              string                `json:"external_domain"`
	TrustProxyHeaders           bool                  `json:"trust_proxy_headers"`
	SSLCertificatePath          string                `json:"ssl_certificate_path"`
	CertificatePassword         CertificateSecret     `json:"certificate_password"`
	SecureConnectionMode        string                `json:"secure_connection_mode"`
	AutomaticPortMapping        bool                  `json:"automatic_port_mapping"`
	MaxVideoStreams             string                `json:"max_video_streams"`
	RemoteStreamingBitrateLimit string                `json:"remote_streaming_bitrate_limit"`
	NetworkRequestProtocol      string                `json:"network_request_protocol"`
	EffectiveStatus             NetworkSettingsStatus `json:"effective_status"`
}

type CertificateSecret struct {
	Configured bool `json:"configured"`
	Masked     bool `json:"masked"`
}

type NetworkSettingsStatus struct {
	Source                     string   `json:"source"`
	RestartRequiredFields      []string `json:"restart_required_fields"`
	FutureRuntimeFields        []string `json:"future_runtime_fields"`
	AutomaticPortMappingActive bool     `json:"automatic_port_mapping_active"`
	Message                    string   `json:"message"`
}

type UpdateNetworkSettingsInput struct {
	LocalNetworks               []string `json:"local_networks"`
	LocalIPAddress              string   `json:"local_ip_address"`
	LocalHTTPPort               int      `json:"local_http_port"`
	LocalHTTPSPort              int      `json:"local_https_port"`
	AllowRemoteAccess           bool     `json:"allow_remote_access"`
	RemoteIPFilter              []string `json:"remote_ip_filter"`
	RemoteIPFilterMode          string   `json:"remote_ip_filter_mode"`
	PublicHTTPPort              int      `json:"public_http_port"`
	PublicHTTPSPort             int      `json:"public_https_port"`
	ExternalDomain              string   `json:"external_domain"`
	TrustProxyHeaders           bool     `json:"trust_proxy_headers"`
	SSLCertificatePath          string   `json:"ssl_certificate_path"`
	CertificatePassword         string   `json:"certificate_password"`
	ClearCertificatePassword    bool     `json:"clear_certificate_password"`
	SecureConnectionMode        string   `json:"secure_connection_mode"`
	AutomaticPortMapping        bool     `json:"automatic_port_mapping"`
	MaxVideoStreams             string   `json:"max_video_streams"`
	RemoteStreamingBitrateLimit string   `json:"remote_streaming_bitrate_limit"`
	NetworkRequestProtocol      string   `json:"network_request_protocol"`
}

func (s *Service) GetNetworkSettings(ctx context.Context) (NetworkSettings, error) {
	values, err := s.loadCategoryValues(ctx, networkCategory)
	if err != nil {
		return NetworkSettings{}, err
	}
	settings := defaultNetworkSettings()
	settings.LocalNetworks = parseListValue(values, networkLocalNetworksKey, settings.LocalNetworks)
	settings.LocalIPAddress = strings.TrimSpace(values[networkLocalIPAddressKey])
	settings.LocalHTTPPort = parsePortValue(values[networkLocalHTTPPortKey], settings.LocalHTTPPort)
	settings.LocalHTTPSPort = parsePortValue(values[networkLocalHTTPSPortKey], settings.LocalHTTPSPort)
	settings.AllowRemoteAccess = parseBoolValue(values[networkAllowRemoteAccessKey], settings.AllowRemoteAccess)
	settings.RemoteIPFilter = parseListValue(values, networkRemoteIPFilterKey, settings.RemoteIPFilter)
	settings.RemoteIPFilterMode = stringOrDefault(values[networkRemoteIPFilterModeKey], settings.RemoteIPFilterMode)
	settings.PublicHTTPPort = parsePortValue(values[networkPublicHTTPPortKey], settings.PublicHTTPPort)
	settings.PublicHTTPSPort = parsePortValue(values[networkPublicHTTPSPortKey], settings.PublicHTTPSPort)
	settings.ExternalDomain = strings.TrimSpace(values[networkExternalDomainKey])
	settings.TrustProxyHeaders = parseBoolValue(values[networkTrustProxyHeadersKey], settings.TrustProxyHeaders)
	settings.SSLCertificatePath = strings.TrimSpace(values[networkSSLCertificatePathKey])
	settings.CertificatePassword.Configured = strings.TrimSpace(values[networkCertificatePasswordKey]) != ""
	settings.CertificatePassword.Masked = settings.CertificatePassword.Configured
	settings.SecureConnectionMode = stringOrDefault(values[networkSecureConnectionModeKey], settings.SecureConnectionMode)
	settings.AutomaticPortMapping = parseBoolValue(values[networkAutomaticPortMappingKey], settings.AutomaticPortMapping)
	settings.MaxVideoStreams = stringOrDefault(values[networkMaxVideoStreamsKey], settings.MaxVideoStreams)
	settings.RemoteStreamingBitrateLimit = stringOrDefault(values[networkRemoteStreamingBitrateLimitKey], settings.RemoteStreamingBitrateLimit)
	settings.NetworkRequestProtocol = stringOrDefault(values[networkRequestProtocolKey], settings.NetworkRequestProtocol)
	settings.EffectiveStatus = networkEffectiveStatus(len(values) > 0)
	return settings, nil
}

func (s *Service) UpdateNetworkSettings(ctx context.Context, input UpdateNetworkSettingsInput) (NetworkSettings, error) {
	normalized, err := normalizeNetworkSettingsInput(input)
	if err != nil {
		return NetworkSettings{}, err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range map[string]string{
			networkLocalNetworksKey:               strings.Join(normalized.LocalNetworks, "\n"),
			networkLocalIPAddressKey:              normalized.LocalIPAddress,
			networkLocalHTTPPortKey:               strconv.Itoa(normalized.LocalHTTPPort),
			networkLocalHTTPSPortKey:              strconv.Itoa(normalized.LocalHTTPSPort),
			networkAllowRemoteAccessKey:           strconv.FormatBool(normalized.AllowRemoteAccess),
			networkRemoteIPFilterKey:              strings.Join(normalized.RemoteIPFilter, "\n"),
			networkRemoteIPFilterModeKey:          normalized.RemoteIPFilterMode,
			networkPublicHTTPPortKey:              strconv.Itoa(normalized.PublicHTTPPort),
			networkPublicHTTPSPortKey:             strconv.Itoa(normalized.PublicHTTPSPort),
			networkExternalDomainKey:              normalized.ExternalDomain,
			networkTrustProxyHeadersKey:           strconv.FormatBool(normalized.TrustProxyHeaders),
			networkSSLCertificatePathKey:          normalized.SSLCertificatePath,
			networkSecureConnectionModeKey:        normalized.SecureConnectionMode,
			networkAutomaticPortMappingKey:        strconv.FormatBool(normalized.AutomaticPortMapping),
			networkMaxVideoStreamsKey:             normalized.MaxVideoStreams,
			networkRemoteStreamingBitrateLimitKey: normalized.RemoteStreamingBitrateLimit,
			networkRequestProtocolKey:             normalized.NetworkRequestProtocol,
		} {
			if err := upsertCategorySettingWithDB(ctx, tx, networkCategory, key, value, false); err != nil {
				return err
			}
		}
		if normalized.ClearCertificatePassword {
			return deleteCategorySettingWithDB(ctx, tx, networkCategory, networkCertificatePasswordKey)
		}
		if strings.TrimSpace(normalized.CertificatePassword) != "" {
			return upsertCategorySettingWithDB(ctx, tx, networkCategory, networkCertificatePasswordKey, strings.TrimSpace(normalized.CertificatePassword), true)
		}
		return nil
	})
	if err != nil {
		return NetworkSettings{}, err
	}
	return s.GetNetworkSettings(ctx)
}

func defaultNetworkSettings() NetworkSettings {
	return NetworkSettings{
		LocalNetworks:               []string{"192.168.1.0/24", "10.0.0.0/8"},
		RemoteIPFilter:              []string{},
		LocalHTTPPort:               8096,
		LocalHTTPSPort:              8920,
		AllowRemoteAccess:           true,
		RemoteIPFilterMode:          "allow",
		PublicHTTPPort:              8096,
		PublicHTTPSPort:             8920,
		SecureConnectionMode:        "disabled",
		MaxVideoStreams:             "unlimited",
		RemoteStreamingBitrateLimit: "unlimited",
		NetworkRequestProtocol:      "auto",
		EffectiveStatus:             networkEffectiveStatus(false),
	}
}

func normalizeNetworkSettingsInput(input UpdateNetworkSettingsInput) (UpdateNetworkSettingsInput, error) {
	input.LocalNetworks = normalizeStringList(input.LocalNetworks)
	input.RemoteIPFilter = normalizeStringList(input.RemoteIPFilter)
	input.LocalIPAddress = strings.TrimSpace(input.LocalIPAddress)
	input.RemoteIPFilterMode = strings.TrimSpace(input.RemoteIPFilterMode)
	input.ExternalDomain = strings.TrimSpace(input.ExternalDomain)
	input.SSLCertificatePath = strings.TrimSpace(input.SSLCertificatePath)
	input.SecureConnectionMode = strings.TrimSpace(input.SecureConnectionMode)
	input.MaxVideoStreams = strings.TrimSpace(input.MaxVideoStreams)
	input.RemoteStreamingBitrateLimit = strings.TrimSpace(input.RemoteStreamingBitrateLimit)
	input.NetworkRequestProtocol = strings.TrimSpace(input.NetworkRequestProtocol)

	if err := validateAddressList("local_networks", input.LocalNetworks); err != nil {
		return UpdateNetworkSettingsInput{}, err
	}
	if err := validateAddressList("remote_ip_filter", input.RemoteIPFilter); err != nil {
		return UpdateNetworkSettingsInput{}, err
	}
	if input.LocalIPAddress != "" {
		if _, err := netip.ParseAddr(input.LocalIPAddress); err != nil {
			return UpdateNetworkSettingsInput{}, fmt.Errorf("local_ip_address must be a valid IP address")
		}
	}
	for field, port := range map[string]int{
		"local_http_port":   input.LocalHTTPPort,
		"local_https_port":  input.LocalHTTPSPort,
		"public_http_port":  input.PublicHTTPPort,
		"public_https_port": input.PublicHTTPSPort,
	} {
		if port < 1 || port > 65535 {
			return UpdateNetworkSettingsInput{}, fmt.Errorf("%s must be between 1 and 65535", field)
		}
	}
	if !allowedValue(input.RemoteIPFilterMode, "allow", "block") {
		return UpdateNetworkSettingsInput{}, fmt.Errorf("remote_ip_filter_mode must be allow or block")
	}
	if !allowedValue(input.SecureConnectionMode, "disabled", "preferred", "required") {
		return UpdateNetworkSettingsInput{}, fmt.Errorf("secure_connection_mode must be disabled, preferred, or required")
	}
	if !allowedValue(input.MaxVideoStreams, "unlimited", "1", "2", "4", "8") {
		return UpdateNetworkSettingsInput{}, fmt.Errorf("max_video_streams must be unlimited, 1, 2, 4, or 8")
	}
	if !allowedValue(input.RemoteStreamingBitrateLimit, "unlimited", "4mbps", "8mbps", "12mbps", "20mbps") {
		return UpdateNetworkSettingsInput{}, fmt.Errorf("remote_streaming_bitrate_limit must be unlimited, 4mbps, 8mbps, 12mbps, or 20mbps")
	}
	if !allowedValue(input.NetworkRequestProtocol, "auto", "ipv4", "ipv6") {
		return UpdateNetworkSettingsInput{}, fmt.Errorf("network_request_protocol must be auto, ipv4, or ipv6")
	}
	return input, nil
}

func validateAddressList(field string, values []string) error {
	for _, value := range values {
		if _, err := netip.ParsePrefix(value); err == nil {
			continue
		}
		if _, err := netip.ParseAddr(value); err == nil {
			continue
		}
		return fmt.Errorf("%s contains invalid IP or CIDR entry %q", field, value)
	}
	return nil
}

func networkEffectiveStatus(saved bool) NetworkSettingsStatus {
	source := "defaults"
	if saved {
		source = "database"
	}
	return NetworkSettingsStatus{
		Source: source,
		RestartRequiredFields: []string{
			"local_http_port",
			"local_https_port",
			"ssl_certificate_path",
			"secure_connection_mode",
		},
		FutureRuntimeFields: []string{
			"automatic_port_mapping",
			"max_video_streams",
			"remote_streaming_bitrate_limit",
			"network_request_protocol",
		},
		AutomaticPortMappingActive: false,
		Message:                    "Settings are saved configuration; listener, TLS, port mapping, and streaming limit changes may require restart or future runtime support before taking effect.",
	}
}

func normalizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseListValue(values map[string]string, key string, fallback []string) []string {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	return normalizeStringList(strings.Split(value, "\n"))
}

func parsePortValue(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 || parsed > 65535 {
		return fallback
	}
	return parsed
}

func parseBoolValue(value string, fallback bool) bool {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func stringOrDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func allowedValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
