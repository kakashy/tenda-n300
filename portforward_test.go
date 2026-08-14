package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetNAT(t *testing.T) {
	srv := newTestServer(t, true)
	defer srv.Close()

	client, err := NewRouterClient(strings.TrimPrefix(srv.URL, "http://"), "admin")
	if err != nil {
		t.Fatal(err)
	}

	nat, err := client.GetNAT()
	if err != nil {
		t.Fatal(err)
	}
	if len(nat.PortRules) != 2 {
		t.Fatalf("expected 2 port rules, got %d", len(nat.PortRules))
	}

	first := nat.PortRules[0]
	if first.InternalIP != "192.168.1.10" || first.InternalPort != "22" || first.ExternalPort != "22" || first.Protocol != "both" {
		t.Errorf("unexpected first rule: %+v", first)
	}
	if nat.Lan == nil || nat.Lan.IP != "192.168.1.1" || nat.Lan.Mask != "255.255.255.0" {
		t.Errorf("unexpected lan config: %+v", nat.Lan)
	}
}

func TestGetNATEmptyPortList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
		case "/goform/getNAT":
			// portList absent entirely, like a router with no rules
			w.Write([]byte(`{"lanCfg":{"lanIP":"192.168.1.1","lanMask":"255.255.255.0"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewRouterClient(strings.TrimPrefix(srv.URL, "http://"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	nat, err := client.GetNAT()
	if err != nil {
		t.Fatal(err)
	}
	if nat.PortRules == nil || len(nat.PortRules) != 0 {
		t.Fatalf("expected empty non-nil rules, got %#v", nat.PortRules)
	}
}

func TestSetPortForwardRules(t *testing.T) {
	var gotModule2, gotPortList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
		case "/goform/setNAT":
			gotModule2 = r.PostFormValue("module2")
			gotPortList = r.PostFormValue("portList")
			json.NewEncoder(w).Encode(setNATResponse{ErrCode: "0"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewRouterClient(strings.TrimPrefix(srv.URL, "http://"), "admin")
	if err != nil {
		t.Fatal(err)
	}

	rules := []PortForwardRule{
		{InternalIP: "192.168.1.10", InternalPort: "22", ExternalPort: "22", Protocol: "both"},
		{InternalIP: "192.168.1.12", InternalPort: "8080", ExternalPort: "8080", Protocol: "udp"},
	}
	if err := client.SetPortForwardRules(rules); err != nil {
		t.Fatal(err)
	}
	if gotModule2 != "portList" {
		t.Errorf("expected module2=portList, got %q", gotModule2)
	}
	want := "192.168.1.10;22;22;both~192.168.1.12;8080;8080;udp"
	if gotPortList != want {
		t.Errorf("portList = %q, want %q", gotPortList, want)
	}
}

func TestSetPortForwardRulesRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
		case "/goform/setNAT":
			json.NewEncoder(w).Encode(setNATResponse{ErrCode: "1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewRouterClient(strings.TrimPrefix(srv.URL, "http://"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	err = client.SetPortForwardRules([]PortForwardRule{{InternalIP: "192.168.1.10", InternalPort: "22", ExternalPort: "22", Protocol: "both"}})
	if err == nil || !strings.Contains(err.Error(), "errCode: 1") {
		t.Fatalf("expected errCode 1 error, got %v", err)
	}
}

func TestSetPortForwardRulesEmpty(t *testing.T) {
	// Clearing the whole list must still send module2 so the router knows
	// which module is being set.
	var gotModule2, gotPortList string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/goform/getstok":
			json.NewEncoder(w).Encode(stokResponse{Random: "test123"})
		case "/index.html":
			w.Write([]byte("ok"))
		case "/login/Auth":
			w.Header().Set("Location", "/index.html")
			w.WriteHeader(http.StatusFound)
		case "/goform/setNAT":
			gotModule2 = r.PostFormValue("module2")
			gotPortList = r.PostFormValue("portList")
			json.NewEncoder(w).Encode(setNATResponse{ErrCode: "0"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewRouterClient(strings.TrimPrefix(srv.URL, "http://"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SetPortForwardRules(nil); err != nil {
		t.Fatal(err)
	}
	if gotModule2 != "portList" || gotPortList != "" {
		t.Errorf("empty list: module2=%q portList=%q, want module2=portList portList=\"\"", gotModule2, gotPortList)
	}
}

func TestEncodePortList(t *testing.T) {
	tests := []struct {
		name  string
		rules []PortForwardRule
		want  string
	}{
		{"empty", nil, ""},
		{"single", []PortForwardRule{{InternalIP: "192.168.1.5", InternalPort: "80", ExternalPort: "8080", Protocol: "tcp"}}, "192.168.1.5;80;8080;tcp"},
		{"multiple", []PortForwardRule{
			{InternalIP: "192.168.1.5", InternalPort: "80", ExternalPort: "8080", Protocol: "tcp"},
			{InternalIP: "192.168.1.6", InternalPort: "53", ExternalPort: "53", Protocol: "udp"},
		}, "192.168.1.5;80;8080;tcp~192.168.1.6;53;53;udp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := encodePortList(tc.rules); got != tc.want {
				t.Errorf("encodePortList() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSameSubnet(t *testing.T) {
	tests := []struct {
		ip, lanIP, mask string
		want            bool
	}{
		{"192.168.1.10", "192.168.1.1", "255.255.255.0", true},
		{"192.168.0.10", "192.168.1.1", "255.255.255.0", false},
		{"10.0.0.5", "10.0.0.1", "255.0.0.0", true},
		{"10.1.0.5", "10.0.0.1", "255.0.0.0", true},
		{"8.8.8.8", "192.168.1.1", "255.255.255.0", false},
		{"not-an-ip", "192.168.1.1", "255.255.255.0", false},
		{"192.168.1.10", "", "255.255.255.0", false},
	}
	for _, tc := range tests {
		if got := sameSubnet(tc.ip, tc.lanIP, tc.mask); got != tc.want {
			t.Errorf("sameSubnet(%q, %q, %q) = %v, want %v", tc.ip, tc.lanIP, tc.mask, got, tc.want)
		}
	}
}

func TestParsePortForwardFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantProtocol string
		wantRest     string
		wantErr      bool
	}{
		{"default protocol", []string{"192.168.1.10", "22", "22"}, "both", "192.168.1.10 22 22", false},
		{"flag after positionals", []string{"192.168.1.10", "22", "22", "--protocol", "tcp"}, "tcp", "192.168.1.10 22 22", false},
		{"flag before positionals", []string{"--protocol", "udp", "192.168.1.10", "22", "22"}, "udp", "192.168.1.10 22 22", false},
		{"equals form", []string{"--protocol=both", "192.168.1.10", "22", "22"}, "both", "192.168.1.10 22 22", false},
		{"missing value", []string{"192.168.1.10", "22", "22", "--protocol"}, "", "", true},
		{"flag-like value", []string{"--protocol", "--json", "192.168.1.10"}, "", "", true},
		{"double dash terminator", []string{"192.168.1.10", "--", "--protocol"}, "both", "192.168.1.10 --protocol", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proto, rest, err := parsePortForwardFlags(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if proto != tc.wantProtocol {
				t.Errorf("protocol = %q, want %q", proto, tc.wantProtocol)
			}
			if got := strings.Join(rest, " "); got != tc.wantRest {
				t.Errorf("rest = %q, want %q", got, tc.wantRest)
			}
		})
	}
}
