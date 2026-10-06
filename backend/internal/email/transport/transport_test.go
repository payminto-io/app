package transport

import "testing"

func TestConfig_Enabled(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"host only", Config{Host: "smtp.example.com"}, false},
		{"host+port no from", Config{Host: "smtp.example.com", Port: 587}, false},
		{"complete", Config{Host: "smtp.example.com", Port: 587, From: "no-reply@x.io"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.Enabled(); got != tc.want {
				t.Errorf("Enabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNew_SelectsSender(t *testing.T) {
	if _, ok := New(Config{}).(*NoopSender); !ok {
		t.Error("expected NoopSender when SMTP not configured")
	}
	full := Config{Host: "smtp.example.com", Port: 587, From: "no-reply@x.io"}
	if _, ok := New(full).(*SMTPSender); !ok {
		t.Error("expected SMTPSender when SMTP configured")
	}
}

func TestNoopSender_DoesNotError(t *testing.T) {
	if err := (&NoopSender{}).Send("a@b.com", "hi", "<p>body</p>"); err != nil {
		t.Errorf("NoopSender.Send returned error: %v", err)
	}
}

func TestBuildMessage_HasHeaders(t *testing.T) {
	msg := string(buildMessage("from@x.io", "to@y.io", "Subj", "<b>hi</b>"))
	for _, want := range []string{"From: from@x.io", "To: to@y.io", "Subject: Subj", "<b>hi</b>"} {
		if !contains(msg, want) {
			t.Errorf("message missing %q\n%s", want, msg)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
