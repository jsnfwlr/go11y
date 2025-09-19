package config

type Severity int

const (
	SeverityLowest  Severity = -2 // No threat to system/process operation - the user can fix this themselves and continue this one operation
	SeverityLow     Severity = -1 // No threat to system/process operation - the user can fix this themselves but will need to restart the operation
	SeverityMedium  Severity = 0  // The error may cause some disruption to system/process operation - the user may be able to fix this themselves but may need support
	SeverityHigh    Severity = 1  // The error will cause disruption to system/process operation - something outside the user's control will need to be fixed
	SeverityHighest Severity = 2  // The error will cause major disruption to system/process operation - something outside the user's control will need to be fixed, and there may be wider implications for the system/process as a whole
)
