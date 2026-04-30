package render

// Renderer is the main rendering struct for CLI output.
// It holds a TTY detector for conditional formatting.
// View rendering methods will be added in subsequent tasks.
type Renderer struct {
	TTY TTYDetector
}

// NewRenderer creates a Renderer with the given TTY detector.
func NewRenderer(tty TTYDetector) *Renderer {
	return &Renderer{TTY: tty}
}

// IsTTY returns whether the renderer is connected to a terminal.
func (r *Renderer) IsTTY() bool {
	if r.TTY == nil {
		return false
	}
	return r.TTY.IsTTY()
}
