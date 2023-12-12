package composable

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
