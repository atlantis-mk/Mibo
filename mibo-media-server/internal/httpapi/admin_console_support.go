package httpapi

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

func buildAdminConsoleAccessAddresses(req *http.Request) []adminConsoleAccessAddress {
	baseURL := requestBaseURL(req)
	addresses := []adminConsoleAccessAddress{
		{Kind: "local", Label: "本机访问", URL: baseURL, Status: "available", Copyable: true},
	}
	for _, ip := range lanIPv4Addresses() {
		addresses = append(addresses, adminConsoleAccessAddress{Kind: "lan", Label: "局域网访问", URL: replaceURLHost(baseURL, ip), Status: "available", Copyable: true})
	}
	if len(addresses) == 1 {
		addresses = append(addresses, adminConsoleAccessAddress{Kind: "lan", Label: "局域网访问", Status: "unavailable", Message: "未发现可用局域网地址", Copyable: false})
	}
	addresses = append(addresses, adminConsoleAccessAddress{Kind: "remote", Label: "远程访问", Status: "not_configured", Route: "/settings", Message: "未配置", Copyable: false})
	return addresses
}

func buildAdminConsoleQuickActions() []adminConsoleQuickAction {
	return []adminConsoleQuickAction{
		{ID: "open-settings", Label: "打开设置", Description: "进入现有设置区域", Kind: "route", Route: "/settings", Risk: "safe"},
		{ID: "open-libraries", Label: "媒体库管理", Description: "管理媒体库与来源", Kind: "route", Route: "/settings/library", Risk: "safe"},
		{ID: "scan-libraries", Label: "扫描媒体库", Description: "为所有媒体库排队扫描任务", Kind: "mutation", Method: "POST", Endpoint: "/api/v1/admin/console/actions/scan-libraries", Risk: "expensive", Confirm: true},
		{ID: "open-logs", Label: "查看日志", Description: "日志查看尚未实现", Kind: "unsupported", Disabled: true, DisabledReason: "日志页面尚未实现", Risk: "safe"},
		{ID: "shutdown", Label: "关闭服务器", Description: "服务器生命周期控制尚未实现", Kind: "unsupported", Disabled: true, DisabledReason: "未提供安全关闭接口", Risk: "danger"},
	}
}

func boolStatus(enabled bool) string {
	if enabled {
		return "ok"
	}
	return "unavailable"
}

func enabledMessage(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func configuredPort(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		trimmed := strings.TrimPrefix(strings.TrimSpace(addr), ":")
		parsed, _ := strconv.Atoi(trimmed)
		return parsed
	}
	parsed, _ := strconv.Atoi(port)
	return parsed
}

func replaceURLHost(baseURL, host string) string {
	scheme := "http://"
	value := strings.TrimPrefix(baseURL, "http://")
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "https://"
		value = strings.TrimPrefix(baseURL, "https://")
	}
	_, port, err := net.SplitHostPort(value)
	if err != nil || port == "" {
		return scheme + host
	}
	return scheme + net.JoinHostPort(host, port)
}

func lanIPv4Addresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var results []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip4 := ip.To4(); ip4 != nil {
				results = append(results, ip4.String())
			}
		}
	}
	return results
}
