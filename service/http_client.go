package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/network_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"golang.org/x/net/proxy"
)

var (
	httpClient      *http.Client
	proxyClientLock sync.Mutex
	proxyClients    = make(map[string]*http.Client)
)

func checkRedirect(req *http.Request, via []*http.Request) error {
	fetchSetting := system_setting.GetFetchSetting()
	urlStr := req.URL.String()
	if err := common.ValidateURLWithFetchSetting(urlStr, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		return fmt.Errorf("redirect to %s blocked: %v", urlStr, err)
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return nil
}

func InitHttpClient() error {
	client, err := newSharedHTTPClient()
	if err != nil {
		return err
	}
	httpClient = client
	return nil
}

func GetHttpClient() *http.Client {
	if httpClient == nil {
		if err := InitHttpClient(); err != nil {
			return http.DefaultClient
		}
	}
	return httpClient
}

// GetHttpClientWithProxy returns the default client or a proxy-enabled one when proxyURL is provided.
func GetHttpClientWithProxy(proxyURL string) (*http.Client, error) {
	if proxyURL == "" {
		return GetHttpClient(), nil
	}
	return NewProxyHttpClient(proxyURL)
}

func newSharedHTTPClient() (*http.Client, error) {
	proxyURL := ""
	if network_setting.IsProxyEnabled() {
		proxyURL = network_setting.GetProxyURL()
	}
	transport, err := newTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	return newHTTPClient(transport), nil
}

func newHTTPClient(transport *http.Transport) *http.Client {
	client := &http.Client{
		Transport:     transport,
		CheckRedirect: checkRedirect,
	}
	if common.RelayTimeout != 0 {
		client.Timeout = time.Duration(common.RelayTimeout) * time.Second
	}
	return client
}

func newTransport(proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{
		MaxIdleConns:        common.RelayMaxIdleConns,
		MaxIdleConnsPerHost: common.RelayMaxIdleConnsPerHost,
		ForceAttemptHTTP2:   true,
	}
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		switch parsedURL.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(parsedURL)
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if parsedURL.User != nil {
				auth = &proxy.Auth{User: parsedURL.User.Username()}
				if password, ok := parsedURL.User.Password(); ok {
					auth.Password = password
				}
			}
			dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
			if err != nil {
				return nil, err
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		default:
			return nil, fmt.Errorf("unsupported proxy scheme: %s, must be http, https, socks5 or socks5h", parsedURL.Scheme)
		}
	} else {
		// No explicit proxy configured — respect environment variables (original behavior)
		transport.Proxy = http.ProxyFromEnvironment
	}
	if common.TLSInsecureSkipVerify {
		transport.TLSClientConfig = common.InsecureTLSConfig
	}
	return transport, nil
}

// ResetProxyClientCache 清空代理客户端缓存，确保下次使用时重新初始化
func ResetProxyClientCache() {
	proxyClientLock.Lock()
	defer proxyClientLock.Unlock()
	for _, client := range proxyClients {
		if transport, ok := client.Transport.(*http.Transport); ok && transport != nil {
			transport.CloseIdleConnections()
		}
	}
	proxyClients = make(map[string]*http.Client)
}

// NewProxyHttpClient 创建支持代理的 HTTP 客户端
func NewProxyHttpClient(proxyURL string) (*http.Client, error) {
	if proxyURL == "" {
		if client := GetHttpClient(); client != nil {
			return client, nil
		}
		return newSharedHTTPClient()
	}

	proxyClientLock.Lock()
	if client, ok := proxyClients[proxyURL]; ok {
		proxyClientLock.Unlock()
		return client, nil
	}
	proxyClientLock.Unlock()

	transport, err := newTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	client := newHTTPClient(transport)
	proxyClientLock.Lock()
	proxyClients[proxyURL] = client
	proxyClientLock.Unlock()
	return client, nil
}
