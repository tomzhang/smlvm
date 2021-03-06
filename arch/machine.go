package arch

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"time"

	"shanhu.io/smlvm/arch/misc"
	"shanhu.io/smlvm/arch/screen"
	"shanhu.io/smlvm/arch/table"
	"shanhu.io/smlvm/image"
)

// Machine is a multicore shared memory simulated arch8 machine.
type Machine struct {
	phyMem *phyMemory
	inst   inst
	calls  *calls

	devices []device
	console *console
	clicks  *screen.Clicks
	screen  *screen.Screen
	table   *table.Table
	rand    *misc.Rand
	ticker  *ticker
	rom     *rom

	cores *multiCore

	// Sections that are loaded into the machine
	Sections []*image.Section
}

// Default SP settings.
const (
	DefaultSPBase   uint32 = 0x20000
	DefaultSPStride uint32 = 0x2000
)

func makeRand(c *Config) *misc.Rand {
	if c.RandSeed == 0 {
		return misc.NewTimeRand()
	}
	return misc.NewRand(c.RandSeed)
}

// NewMachine creates a machine with memory and cores.
// 0 memSize for full 4GB memory.
func NewMachine(c *Config) *Machine {
	if c.Ncore == 0 {
		c.Ncore = 1
	}
	m := new(Machine)
	m.phyMem = newPhyMemory(c.MemSize)
	m.inst = new(instArch8)
	m.calls = newCalls(m.phyMem.Page(pageRPC), m.phyMem)
	m.cores = newMultiCore(c.Ncore, m.phyMem, m.calls, m.inst)

	// hook-up devices
	p := m.phyMem.Page(pageBasicIO)

	m.console = newConsole(p, m.cores)
	m.ticker = newTicker(m.cores)

	m.calls.register(serviceConsole, m.console)
	m.calls.register(serviceRand, makeRand(c))
	m.calls.register(serviceClock, &misc.Clock{PerfNow: c.PerfNow})

	m.addDevice(m.ticker)
	m.addDevice(m.console)

	if c.Screen != nil {
		m.clicks = screen.NewClicks(m.calls.sender(serviceScreen))
		s := screen.New(c.Screen)
		m.screen = s
		m.addDevice(s)
		m.calls.register(serviceScreen, s)
	}

	if c.Table != nil {
		t := table.New(c.Table, m.calls.sender(serviceTable))
		m.table = t
		m.calls.register(serviceTable, t) // hook vpc all
	}

	sys := m.phyMem.Page(pageSysInfo)
	sys.WriteWord(0, m.phyMem.npage)
	sys.WriteWord(4, uint32(c.Ncore))

	if c.InitSP == 0 {
		m.cores.setSP(DefaultSPBase, DefaultSPStride)
	} else {
		m.cores.setSP(c.InitSP, c.StackPerCore)
	}
	m.SetPC(c.InitPC)
	if c.Output != nil {
		m.console.Output = c.Output
	}
	if c.ROM != "" {
		m.mountROM(c.ROM)
	}
	if c.RandSeed != 0 {
		m.randSeed(c.RandSeed)
	}
	m.phyMem.WriteWord(AddrBootArg, c.BootArg) // ignoring write error

	return m
}

func (m *Machine) mountROM(root string) {
	p := m.phyMem.Page(pageBasicIO)
	m.rom = newROM(p, m.phyMem, m.cores, root)
	m.addDevice(m.rom)
}

// ReadWord reads a word from the virtual address space.
func (m *Machine) ReadWord(core byte, virtAddr uint32) (uint32, error) {
	return m.cores.readWord(core, virtAddr)
}

// DumpRegs returns the values of the current registers of a core.
func (m *Machine) DumpRegs(core byte) []uint32 {
	return m.cores.dumpRegs(core)
}

func (m *Machine) addDevice(d device) { m.devices = append(m.devices, d) }

// Tick proceeds the simulation by one tick.
func (m *Machine) Tick() *CoreExcep {
	for _, d := range m.devices {
		d.Tick()
	}
	return m.cores.Tick()
}

// Run simulates nticks. It returns the number of ticks
// simulated without error, and the first met error if any.
func (m *Machine) Run(nticks int) (int, *CoreExcep) {
	n := 0
	for i := 0; nticks == 0 || i < nticks; i++ {
		e := m.Tick()
		n++
		if e != nil {
			m.FlushScreen()
			return n, e
		}
	}

	return n, nil
}

// WriteBytes write a byte buffer to the memory at a particular offset.
func (m *Machine) WriteBytes(r io.Reader, offset uint32) error {
	start := offset % PageSize
	pageBuf := make([]byte, PageSize)
	pn := offset / PageSize
	for {
		p := m.phyMem.Page(pn)
		if p == nil {
			return newOutOfRange(offset)
		}

		buf := pageBuf[:PageSize-start]
		n, err := r.Read(buf)
		if err == io.EOF {
			return nil
		}

		p.WriteAt(buf[:n], start)
		start = 0
		pn++
	}

	return nil
}

func (m *Machine) randSeed(s int64) {
	m.ticker.Rand = rand.New(rand.NewSource(s))
}

// LoadSections loads a list of sections into the machine.
func (m *Machine) LoadSections(secs []*image.Section) error {
	for _, s := range secs {
		var buf io.Reader
		switch s.Type {
		case image.Zeros:
			buf = &zeroReader{s.Header.Size}
		case image.Code, image.Data:
			buf = bytes.NewReader(s.Bytes)
		case image.None, image.Debug, image.Comment:
			continue
		default:
			return fmt.Errorf("unknown section type: %d", s.Type)
		}

		if err := m.WriteBytes(buf, s.Addr); err != nil {
			return err
		}
	}

	if pc, found := image.CodeStart(secs); found {
		m.SetPC(pc)
	}
	m.Sections = secs

	return nil
}

// SetPC sets all cores to start with a particular PC pointer.
func (m *Machine) SetPC(pc uint32) {
	for _, cpu := range m.cores.cores {
		cpu.regs[PC] = pc
	}
}

// LoadImage loads an e8 image into the machine.
func (m *Machine) LoadImage(r io.ReadSeeker) error {
	secs, err := image.Read(r)
	if err != nil {
		return err
	}
	return m.LoadSections(secs)
}

// LoadImageBytes loads an e8 image in bytes into the machine.
func (m *Machine) LoadImageBytes(bs []byte) error {
	return m.LoadImage(bytes.NewReader(bs))
}

// PrintCoreStatus prints the cpu statuses.
func (m *Machine) PrintCoreStatus() { m.cores.PrintStatus() }

// FlushScreen flushes updates in the frame buffer to the
// screen device, even if the device has not asked for an update.
func (m *Machine) FlushScreen() {
	if m.screen != nil {
		m.screen.Flush()
	}
}

// Click sends in a mouse click at the particular location.
func (m *Machine) Click(line, col uint8) {
	if m.clicks == nil {
		return
	}
	m.clicks.Click(line, col)
}

// ClickTable sends a click on the table at the particular location.
func (m *Machine) ClickTable(pos uint8) {
	if m.table == nil {
		return
	}
	m.table.Click(pos)
}

// SleepTime returns the sleeping time required before next execution.
func (m *Machine) SleepTime() (time.Duration, bool) {
	return m.calls.sleepTime()
}

// HasPending checks if the machine has pending messages that are not
// delivered.
func (m *Machine) HasPending() bool {
	return m.calls.queueLen() > 0
}
