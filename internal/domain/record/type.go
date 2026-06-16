package record

// Type classifies collaboration text attached to a task transition.
type Type string

const (
	TypeSubmit     Type = "submit"
	TypeReject     Type = "reject"
	TypeApprove    Type = "approve"
	TypeCancel     Type = "cancel"
	TypeReactivate Type = "reactivate"
)

func (t Type) String() string {
	return string(t)
}
