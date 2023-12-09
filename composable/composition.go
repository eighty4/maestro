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

type Composable interface {
	Start()
	Restart()
	Status() CompositionStatus
	Stop()
}

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
