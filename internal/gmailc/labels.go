package gmailc

import "google.golang.org/api/gmail/v1"

func (c *Client) EnsureLabelExists(name string) (string, error) {
	res, err := c.Srv.Users.Labels.List("me").Do()
	if err != nil {
		return "", err
	}

	for _, l := range res.Labels {
		if l.Name == name {
			return l.Id, nil
		}
	}

	newLabel, err := c.Srv.Users.Labels.Create("me", &gmail.Label{
		Name: name,
	}).Do()
	if err != nil {
		return "", err
	}
	return newLabel.Id, nil
}

func (c *Client) AddLabelToMessage(msgID, labelID string) error {
	_, err := c.Srv.Users.Messages.Modify("me", msgID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{labelID},
	}).Do()
	return err
}
