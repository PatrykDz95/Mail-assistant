package email

type Classification struct {
	Category   Category
	Label      Label
	Reply      string
	SenderName string
}

func NewClassification(category Category, label Label, reply, senderName string) *Classification {
	return &Classification{
		Category:   category,
		Label:      label,
		Reply:      reply,
		SenderName: senderName,
	}
}
