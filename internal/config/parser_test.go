package config

import "testing"

func TestParseSSHCommand(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		ok   bool
	}{
		{"basic local", `ssh -L 8080:localhost:80 user@host`, true},
		{"local+remote+dynamic", `ssh -L 8080:localhost:80 -R 9090:localhost:90 -D 1080 -p 2222 -i ~/.ssh/id_rsa user@host`, true},
		{"bind addr", `ssh -L 127.0.0.1:8080:example.com:80 user@host`, true},
		{"no host", `ssh -L 8080:localhost:80`, false},
		{"no user", `ssh -L 8080:localhost:80 host`, false},
		{"no forward", `ssh user@host`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseSSHCommand(c.cmd)
			if c.ok && err != nil {
				t.Fatalf("expected ok, got err: %v", err)
			}
			if !c.ok && err == nil {
				t.Fatalf("expected error, got ok")
			}
		})
	}
}

func TestParseForwardLocal(t *testing.T) {
	tun, err := ParseSSHCommand(`ssh -L 8080:localhost:80 user@host`)
	if err != nil {
		t.Fatal(err)
	}
	if len(tun.Forwards) != 1 {
		t.Fatalf("want 1 forward, got %d", len(tun.Forwards))
	}
	f := tun.Forwards[0]
	if f.Type != ForwardLocal || f.Listen != "8080" || f.Target != "localhost:80" {
		t.Errorf("bad forward: %+v", f)
	}
}

func TestParseForwardRemoteBind(t *testing.T) {
	tun, err := ParseSSHCommand(`ssh -R 0.0.0.0:9090:localhost:90 user@host`)
	if err != nil {
		t.Fatal(err)
	}
	f := tun.Forwards[0]
	if f.Type != ForwardRemote || f.Listen != "0.0.0.0:9090" || f.Target != "localhost:90" {
		t.Errorf("bad forward: %+v", f)
	}
}
