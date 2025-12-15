package email

type Label string

const (
	LabelNewsletter   Label = "newsletter"
	LabelPrivate      Label = "private"
	LabelBusiness     Label = "business"
	LabelPayments     Label = "payments"
	LabelActionNeeded Label = "action_needed"
	LabelJunk         Label = "junk"
)

type Category string // TODO: can be merged with Label?

const (
	CategoryNewsletter   Category = "newsletter"
	CategoryPrivate      Category = "private"
	CategoryBusiness     Category = "business"
	CategoryPayments     Category = "payments"
	CategoryActionNeeded Category = "action_needed"
	CategoryJunk         Category = "junk"
)

func (l Label) IsValid() bool {
	switch l {
	case LabelNewsletter, LabelPrivate, LabelBusiness, LabelPayments, LabelActionNeeded, LabelJunk:
		return true
	}
	return false
}

func (l Label) String() string {
	return string(l)
}
