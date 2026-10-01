package go11y

// SeverityLowest errors pose no threat to system/process operation - the user can fix this themselves and continue this
// one operation
const SeverityLowest string = "lowest"

// SeverityLow errors pose no threat to system/process operation - the user can fix this themselves but will need to
// restart the operation
const SeverityLow string = "low"

// SeverityMedium errors may cause some disruption to system/process operation - the user may be able to fix this
// themselves but may need support
const SeverityMedium string = "medium"

// SeverityHigh errors will cause disruption to system/process operation - something outside the user's control will
// need to be fixed
const SeverityHigh string = "high"

// SeverityHighest errors will cause major disruption to system/process operation - something outside the user's control
//
//	will need to be fixed, and there may be wider implications for the system/process as a whole
const SeverityHighest string = "highest"
