package gmailc

import "google.golang.org/api/gmail/v1"

type Client struct {
	Srv *gmail.Service
}
