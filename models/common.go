package models

const (
	STATE_EXIST   = 1
	STATE_DELETED = 2
)

type ConfirmationState int64

const (
	ConfirmationStateUnconfirmed ConfirmationState = 1
	ConfirmationStateConfirmed   ConfirmationState = 2
)
