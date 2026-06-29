package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, loginOK bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			if loginOK {
				w.Header().Set("Location", "/index.html")
				w.WriteHeader(http.StatusFound)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"errCode":"1"}`))
			}
		case "/goform/getQos":
			json.NewEncoder(w).Encode(qosResponse{
				OnlineList: []qosDevice{
					{Hostname: "phone", MAC: "aa:bb:cc:dd:ee:01", IP: "192.168.1.10", ConnectType: "WiFi", Access: "true"},
					{Hostname: "laptop", MAC: "aa:bb:cc:dd:ee:02", IP: "192.168.1.11", ConnectType: "WiFi", Access: "true"},
				},
				BlackList: []qosDevice{
					{Hostname: "blocked-device", MAC: "aa:bb:cc:dd:ee:ff", IP: "0.0.0.0", ConnectType: "WiFi", Access: "false"},
				},
			})
		case "/goform/setQos":
			var res setQosResponse
			body, _ := json.Marshal(r.PostFormValue("qosList"))
			if strings.Contains(string(body), "target") {
				res.ErrCode = "0"
			} else {
				// accept anything with at least one entry
				if r.PostFormValue("qosList") != "" {
					res.ErrCode = "0"
				} else {
					res.ErrCode = "1"
				}
			}
			json.NewEncoder(w).Encode(res)
		case "/goform/sysReboot":
			w.WriteHeader(http.StatusOK)
		case "/goform/sysRestore":
			w.WriteHeader(http.StatusOK)
		case "/cgi-bin/DownloadCfg/RouterCfm.cfg":
			w.Write([]byte("mock-config-data"))
		case "/cgi-bin/DownloadSyslog/RouterSystem.log":
			w.Write([]byte("mock-syslog-data"))
		case "/cgi-bin/UploadCfg":
			w.Write([]byte(`{"errCode":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestNewRouterClient(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewRouterClientLoginFails(t *testing.T) {
	srv := newTestServer(t, false)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	_, err := NewRouterClient(ip, "wrong")
	if err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestGetDevices(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	devices, err := client.GetDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}

	expected := map[string]bool{
		"aa:bb:cc:dd:ee:01": true,
		"aa:bb:cc:dd:ee:02": true,
		"aa:bb:cc:dd:ee:ff": false,
	}
	for _, d := range devices {
		access, ok := expected[d.MAC]
		if !ok {
			t.Errorf("unexpected device MAC: %s", d.MAC)
			continue
		}
		if d.Access != access {
			t.Errorf("device %s: expected access=%v, got %v", d.MAC, access, d.Access)
		}
	}
}

func TestBlockMAC(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.BlockMAC("aa:bb:cc:dd:ee:01"); err != nil {
		t.Fatal(err)
	}
}

func TestUnblockMAC(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.UnblockMAC("aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
}

func TestReboot(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Reboot(); err != nil {
		t.Fatal(err)
	}
}

func TestReset(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Reset(); err != nil {
		t.Fatal(err)
	}
}

func TestBackupConfig(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.BackupConfig()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mock-config-data" {
		t.Fatalf("expected mock-config-data, got %s", string(data))
	}
}

func TestExportSyslog(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.ExportSyslog()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mock-syslog-data" {
		t.Fatalf("expected mock-syslog-data, got %s", string(data))
	}
}

func TestGetDevicesJSONParse(t *testing.T) {
	// Test the JSON structure directly to catch field name drift
	raw := `{
		"onlineList": [
			{"qosListHostname": "pc", "qosListMac": "11:22:33:44:55:66", "qosListIP": "10.0.0.2", "qosListConnectType": "Ethernet", "qosListUpLimit": "", "qosListDownLimit": "", "qosListAccess": "true", "qosListRemark": ""}
		],
		"blackList": []
	}`
	var qos qosResponse
	if err := json.Unmarshal([]byte(raw), &qos); err != nil {
		t.Fatal(err)
	}
	if len(qos.OnlineList) != 1 {
		t.Fatalf("expected 1 online device, got %d", len(qos.OnlineList))
	}
	if qos.OnlineList[0].MAC != "11:22:33:44:55:66" {
		t.Fatalf("expected MAC 11:22:33:44:55:66, got %s", qos.OnlineList[0].MAC)
	}
}

func TestSetQosResponse(t *testing.T) {
	tests := []struct {
		json string
		want string
	}{
		{`{"errCode":"0"}`, "0"},
		{`{"errCode":"1"}`, "1"},
	}
	for _, tc := range tests {
		var res setQosResponse
		if err := json.Unmarshal([]byte(tc.json), &res); err != nil {
			t.Fatal(err)
		}
		if res.ErrCode != tc.want {
			t.Fatalf("expected errCode %s, got %s", tc.want, res.ErrCode)
		}
	}
}

func TestRestoreConfig(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	ip := strings.TrimPrefix(srv.URL, "http://")
	client, err := NewRouterClient(ip, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestoreConfig("router_test.go"); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultGatewayIP(t *testing.T) {
	ip, err := defaultGatewayIP()
	if err != nil {
		t.Skipf("no default gateway: %v", err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("invalid IP from defaultGatewayIP: %s", ip)
	}
}
