package composable

// CompositionStatus represents states of a Composition.
type CompositionStatus string

const (
	CompositionNotStarted CompositionStatus = "NotStarted"
	CompositionStarting   CompositionStatus = "Starting"
	CompositionRunning    CompositionStatus = "Running"
	CompositionFailing    CompositionStatus = "Failing"
	CompositionError      CompositionStatus = "Error"
	CompositionStopped    CompositionStatus = "Stopped"
)

type Composition struct {
	composables []Composable
}

func NewComposition(composables []Composable) *Composition {
	return &Composition{
		composables: composables,
	}
}

func (c *Composition) Start() {
	for _, composable := range c.composables {
		composable.Start()
	}
}

type Composable interface {
	Start()
	Restart()
	Status() CompositionStatus
	Stop()
}

type ExecComposable struct {
	exec    *ExecDescription
	process *Process
}

func NewExecComposable(exec *ExecDescription) *ExecComposable {
	return &ExecComposable{
		exec: exec,
	}
}

func (c ExecComposable) Start() {
	if c.process != nil {
		c.Stop()
	}
	c.process = c.exec.Process()
	c.process.Start()
}

func (c ExecComposable) Restart() {
	if c.process == nil {
		c.Start()
	} else {
		c.process.Restart()
	}
}

func (c ExecComposable) Status() CompositionStatus {
	if c.process != nil {
		switch c.process.Status {
		case ProcessNotStarted:
			return CompositionNotStarted
		case ProcessStopped:
			return CompositionStopped
		case ProcessError:
			return CompositionError
		case ProcessRunning:
			return CompositionRunning
		}
	}
	return CompositionNotStarted
}

func (c ExecComposable) Stop() {
	if c.process != nil {
		c.process.Stop()
	}
}
