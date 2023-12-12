package composable

// Status represents states of a Composable.
type Status string

const (
	// NotStarted is the status of a Composable on creation before calling Composable.Start.
	NotStarted Status = "NotStarted"
	// Starting is the status of a running Composable.
	Starting Status = "Starting"
	// Running is the status of a running Composable.
	Running Status = "Running"
	// Error is the status of a Composable that has failed to start or exited with an error exit code.
	Error Status = "Error"
	// Stopped is the status of a stopped Composable.
	Stopped Status = "Stopped"
)

type Composable interface {
	Start()
	Restart()
	Status() Status
	StatusC() <-chan Status
	Stop()
}
