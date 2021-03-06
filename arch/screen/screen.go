package screen

import (
	"shanhu.io/smlvm/arch/vpc"
	"shanhu.io/smlvm/coder"
)

// Screen is the screen buffer.
type Screen struct {
	textUpdate  map[uint32]byte
	colorUpdate map[uint32]byte
	r           Render
}

// New creates a new screen that renders on the given renderer.
func New(r Render) *Screen {
	if r == nil {
		panic("creating nil screen")
	}

	return &Screen{
		textUpdate:  make(map[uint32]byte),
		colorUpdate: make(map[uint32]byte),
		r:           r,
	}
}

// Handle handles incoming requirement.
func (s *Screen) Handle(req []byte) ([]byte, int32) {
	dec := coder.NewDecoder(req)
	cmd := dec.U8()
	if dec.Err != nil {
		return nil, vpc.ErrInvalidArg
	}

	switch cmd {
	case 0, 1:
		c := dec.U8()
		line := uint32(dec.U8())
		col := uint32(dec.U8())

		if dec.Err != nil {
			return nil, vpc.ErrInvalidArg
		}

		if cmd == 0 {
			s.textUpdate[line*Width+col] = c
		} else { // cmd == 1
			s.colorUpdate[line*Width+col] = c
		}
	default:
		return nil, vpc.ErrInvalidArg
	}

	return nil, 0
}

// Flush flushes the screen buffer.
func (s *Screen) Flush() {
	if len(s.textUpdate) > 0 {
		s.r.UpdateText(s.textUpdate)
		s.textUpdate = make(map[uint32]byte)
	}
	if len(s.colorUpdate) > 0 {
		s.r.UpdateColor(s.colorUpdate)
		s.colorUpdate = make(map[uint32]byte)
	}
}

// Tick flushes the screen if it needs to.
func (s *Screen) Tick() {
	if s.r.NeedUpdate() {
		s.Flush()
	}
}
