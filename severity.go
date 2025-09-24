package go11y

const (
	SeverityLowest  string = "lowest"  // No threat to system/process operation - the user can fix this themselves and continue this one operation
	SeverityLow     string = "low"     // No threat to system/process operation - the user can fix this themselves but will need to restart the operation
	SeverityMedium  string = "medium"  // The error may cause some disruption to system/process operation - the user may be able to fix this themselves but may need support
	SeverityHigh    string = "high"    // The error will cause disruption to system/process operation - something outside the user's control will need to be fixed
	SeverityHighest string = "highest" // The error will cause major disruption to system/process operation - something outside the user's control will need to be fixed, and there may be wider implications for the system/process as a whole
)
