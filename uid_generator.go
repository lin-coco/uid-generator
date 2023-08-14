package uidgenerator

// UidGenerator Represents a unique id generator.
type UidGenerator interface {
	// GetUID Get a unique ID
	GetUID() (int64, error)

	// ParseUID Parse the UID into elements which are used to generate the UID. <br>
	// Such as timestamp & workerId & sequence...
	ParseUID(uid int64) string
}
