package outlook_http

import "testing"

func TestNormalizeBody(t *testing.T) {
	cases := []struct {
		name string
		in   Body
		want string
	}{
		// Body measured on 2026-10-30 for an event synced with an empty description.
		{"empty converted html", Body{ContentType: "html", Content: "<html>\r\n<head>\r\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\r\n</head>\r\n<body>\r\n<!-- converted from text -->\r\n<div class=\"PlainText\">&nbsp;</div>\r\n</body>\r\n</html>\r\n"}, ""},
		{"empty text", Body{ContentType: "text", Content: ""}, ""},
		{"text with nbsp", Body{ContentType: "text", Content: " \r\n"}, ""},
		{"text crlf", Body{ContentType: "text", Content: "ligne 1\r\nligne 2\r\n"}, "ligne 1\nligne 2"},
		{"html text", Body{ContentType: "html", Content: "<div class=\"PlainText\">Salle A &amp; B</div>"}, "Salle A & B"},
	}
	for _, c := range cases {
		if got := normalizeBody(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
