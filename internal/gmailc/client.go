package gmailc

import (
	"google.golang.org/api/gmail/v1"
	"log"
	"strings"
)

type Client struct {
	Srv      *gmail.Service
	LabelIDs map[string]string
}

func (c *Client) InitLabels() error {
	c.LabelIDs = make(map[string]string)

	// 1) Fetch existing labels
	list, err := c.Srv.Users.Labels.List("me").Do()
	if err != nil {
		return err
	}

	// Fill existing labels
	for _, l := range list.Labels {
		c.LabelIDs[l.Name] = l.Id
	}

	// 2) Ensure required labels exist
	for _, gmailName := range LabelMap {

		// Already known
		if id, ok := c.LabelIDs[gmailName]; ok {
			_ = id
			continue
		}

		// Try create the label
		created, err := c.Srv.Users.Labels.Create("me", &gmail.Label{Name: gmailName}).Do()
		if err != nil {

			// --- CASE: Gmail returns conflict (409) ---
			if strings.Contains(err.Error(), "Label name exists or conflicts") {

				// Gmail says the label exists somewhere; trust it.
				log.Printf("Label %q already exists (409 conflict), continuing", gmailName)
				continue
			}

			// other errors ARE fatal
			return err
		}

		// Creation succeeded → store ID
		c.LabelIDs[gmailName] = created.Id
	}

	return nil
}
