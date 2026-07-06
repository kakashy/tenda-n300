package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Type     string `json:"type"`
	Access   bool   `json:"access"`
}

type qosResponse struct {
	OnlineList []qosDevice `json:"onlineList"`
	BlackList  []qosDevice `json:"blackList"`
}

type qosDevice struct {
	Hostname    string `json:"qosListHostname"`
	Remark      string `json:"qosListRemark"`
	MAC         string `json:"qosListMac"`
	IP          string `json:"qosListIP"`
	ConnectType string `json:"qosListConnectType"`
	UpLimit     string `json:"qosListUpLimit"`
	DownLimit   string `json:"qosListDownLimit"`
	Access      string `json:"qosListAccess"`
}

type stokResponse struct {
	Random string `json:"random"`
}

type FirmwareInfo struct {
	Model          string `json:"model"`
	Version        string `json:"version"`
	Hardware       string `json:"hardware"`
	DefaultDNS     string `json:"defaultdns"`
	AltDNS         string `json:"altdns"`
	ConnectionType string `json:"connectiontype"`
	Gateway        string `json:"gateway"`
}

type internetCfgResponse struct {
	WanType    string `json:"wanType"`
	DNSServer  string `json:"dnsServer"`
	DNSServer1 string `json:"dnsServer1"`
	Gateway    string `json:"gateway"`
}

type setQosResponse struct {
	ErrCode string `json:"errCode"`
}

const clientTimeout = 10 * time.Second

// wrapTimeoutError returns a user-friendly message if err is a network timeout.
func wrapTimeoutError(err error) error {
	if err == nil {
		return nil
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("connection to router timed out (is the router online?)")
	}
	return err
}

type RouterClient struct {
	baseURL string
	client  *http.Client
}

func NewRouterClient(ip, password string) (*RouterClient, error) {
	baseURL := fmt.Sprintf("http://%s", ip)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &RouterClient{
		baseURL: baseURL,
		client:  &http.Client{Jar: jar, Timeout: clientTimeout},
	}
	if err := c.login(password); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RouterClient) getStok() (string, error) {
	resp, err := c.client.Get(c.baseURL + "/goform/getstok")
	if err != nil {
		return "", wrapTimeoutError(fmt.Errorf("getstok: %w", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var s stokResponse
	if err := json.Unmarshal(body, &s); err != nil {
		return "", fmt.Errorf("decode stok: %w", err)
	}
	return s.Random, nil
}

func (c *RouterClient) login(password string) error {
	for i := 0; i < 3; i++ {
		random, err := c.getStok()
		if err != nil {
			if i == 2 {
				return err
			}
			continue
		}
		b64pass := base64.StdEncoding.EncodeToString([]byte(password))
		hash := md5.Sum([]byte(b64pass + random))
		encoded := hex.EncodeToString(hash[:])
		data := url.Values{"password": {encoded}}
		req, err := http.NewRequest("POST", c.baseURL+"/login/Auth", strings.NewReader(data.Encode()))
		if err != nil {
			if i == 2 {
				return err
			}
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "tenda-n300/1.0")
		req.Header.Set("Referer", c.baseURL+"/login.html")
		resp, err := c.client.Do(req)
		if err != nil {
			if i == 2 {
				return wrapTimeoutError(fmt.Errorf("login: %w", err))
			}
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if strings.Contains(resp.Request.URL.String(), "index.html") {
			return nil
		}
	}
	return fmt.Errorf("login failed (wrong password or rate-limited)")
}

func (c *RouterClient) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wrapTimeoutError(err)
	}
	return resp, nil
}

func (c *RouterClient) post(path string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.baseURL+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wrapTimeoutError(err)
	}
	return resp, nil
}

func (c *RouterClient) GetDevices() ([]Device, error) {
	resp, err := c.get("/goform/getQos?modules=onlineList,blackList")
	if err != nil {
		return nil, fmt.Errorf("getQos: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var qos qosResponse
	if err := json.Unmarshal(body, &qos); err != nil {
		return nil, fmt.Errorf("decode QoS: %w", err)
	}
	devices := make([]Device, 0, len(qos.OnlineList)+len(qos.BlackList))
	for _, d := range qos.OnlineList {
		devices = append(devices, Device{
			Hostname: d.Hostname,
			IP:       d.IP,
			MAC:      d.MAC,
			Type:     d.ConnectType,
			Access:   true,
		})
	}
	for _, d := range qos.BlackList {
		devices = append(devices, Device{
			Hostname: d.Hostname,
			IP:       d.IP,
			MAC:      d.MAC,
			Type:     d.ConnectType,
			Access:   false,
		})
	}
	return devices, nil
}

func (c *RouterClient) BlockMAC(mac string) error {
	return c.setMACAccess(mac, "block")
}

func (c *RouterClient) UnblockMAC(mac string) error {
	return c.setMACAccess(mac, "unblock")
}

func (c *RouterClient) setMACAccess(targetMAC, action string) error {
	// Validate and normalize MAC address
	if err := ValidateMAC(targetMAC); err != nil {
		return err
	}
	targetMAC = NormalizeMAC(targetMAC)

	resp, err := c.get("/goform/getQos?modules=onlineList,blackList")
	if err != nil {
		return fmt.Errorf("fetch QoS: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	var qos qosResponse
	if err := json.Unmarshal(body, &qos); err != nil {
		return fmt.Errorf("decode QoS: %w", err)
	}

	var b strings.Builder

	for _, d := range qos.OnlineList {
		mac := NormalizeMAC(d.MAC)
		access := d.Access
		if mac == targetMAC {
			if action == "block" {
				access = "false"
			} else {
				access = "true"
			}
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n", d.Hostname, d.Remark, d.MAC, d.UpLimit, d.DownLimit, access))
	}
	for _, d := range qos.BlackList {
		mac := NormalizeMAC(d.MAC)
		if mac == targetMAC && action == "unblock" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\tfalse\n", d.Hostname, d.Remark, d.MAC, d.UpLimit, d.DownLimit))
	}

	qosList := strings.TrimRight(b.String(), "\n")
	data := url.Values{
		"module1": {"qosList"},
		"qosList": {qosList},
	}
	resp2, err := c.post("/goform/setQos", data)
	if err != nil {
		return fmt.Errorf("setQos: %w", err)
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	var res setQosResponse
	if err := json.Unmarshal(body2, &res); err != nil {
		return fmt.Errorf("decode setQos: %w", err)
	}
	if res.ErrCode != "0" {
		return fmt.Errorf("QoS rejected, errCode: %s", res.ErrCode)
	}
	return nil
}

func (c *RouterClient) Reboot() error {
	data := url.Values{"module1": {"sysOperate"}, "action": {"reboot"}}
	resp, err := c.post("/goform/sysReboot", data)
	if err != nil {
		return fmt.Errorf("reboot: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *RouterClient) Reset() error {
	data := url.Values{"module1": {"sysOperate"}, "action": {"restore"}}
	resp, err := c.post("/goform/sysRestore", data)
	if err != nil {
		return fmt.Errorf("reset: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *RouterClient) BackupConfig() ([]byte, error) {
	resp, err := c.get("/cgi-bin/DownloadCfg/RouterCfm.cfg")
	if err != nil {
		return nil, fmt.Errorf("backup: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *RouterClient) RestoreConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("inportFile", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/cgi-bin/UploadCfg", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "tenda-n300/1.0")
	req.Header.Set("Referer", c.baseURL+"/index.html")
	resp, err := c.client.Do(req)
	if err != nil {
		return wrapTimeoutError(fmt.Errorf("restore: %w", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("restore: read response: %w", err)
	}

	var result struct {
		ErrCode json.RawMessage `json:"errCode"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("restore: parse response: %w", err)
	}

	code := strings.Trim(string(result.ErrCode), `" `)
	if code == "0" || code == "100" {
		return nil
	}
	return fmt.Errorf("restore failed: server returned unexpected response")
}

func (c *RouterClient) GetFirmwareInfo() *FirmwareInfo {
	info := &FirmwareInfo{}

	body, err := c.getLoginPage()
	if err == nil {
		info.Model = extractJSVar(body, "model_name")
		info.Hardware = extractJSVar(body, "hardware_version")
		info.Version = extractJSVar(body, "firmware_version")
	}

	if cfg, err := c.getInternetCfg(); err == nil {
		info.DefaultDNS = cfg.DNSServer
		info.AltDNS = cfg.DNSServer1
		info.ConnectionType = strings.ToUpper(cfg.WanType)
		info.Gateway = cfg.Gateway
	}

	return info
}

func (c *RouterClient) getLoginPage() ([]byte, error) {
	resp, err := c.get("/login.html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func extractJSVar(html []byte, name string) string {
	s := string(html)
	prefixes := []string{
		`var ` + name + `=`,
		`var ` + name + ` = `,
	}
	for _, prefix := range prefixes {
		i := strings.Index(s, prefix)
		if i == -1 {
			continue
		}
		start := i + len(prefix)
		if start >= len(s) {
			continue
		}
		quote := s[start]
		if quote != '"' && quote != '\'' {
			continue
		}
		end := strings.IndexByte(s[start+1:], byte(quote))
		if end == -1 {
			continue
		}
		return s[start+1 : start+1+end]
	}
	return ""
}

func (c *RouterClient) getInternetCfg() (*internetCfgResponse, error) {
	resp, err := c.get("/goform/getInternetCfg")
	if err != nil {
		return nil, fmt.Errorf("get internet cfg: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read internet cfg: %w", err)
	}
	var cfg internetCfgResponse
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("decode internet cfg: %w", err)
	}
	return &cfg, nil
}

func (c *RouterClient) ExportSyslog() ([]byte, error) {
	resp, err := c.get("/cgi-bin/DownloadSyslog/RouterSystem.log")
	if err != nil {
		return nil, fmt.Errorf("syslog: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
