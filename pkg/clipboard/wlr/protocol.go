package wlr

import (
	"fmt"
	"os"

	wl "deedles.dev/wl/client"
	"deedles.dev/wl/wire"
)

const (
	ZwlrDataControlManagerV1Interface = "zwlr_data_control_manager_v1"
	ZwlrDataControlManagerV1Version   = 2
)

// This interface is a manager that allows creating per-seat data device
// controls.
type ZwlrDataControlManagerV1 struct {

	// OnDelete is called when the object is removed from the tracking
	// system.
	OnDelete func()

	state wire.State
	id    uint32
}

// NewZwlrDataControlManagerV1 returns a newly instantiated ZwlrDataControlManagerV1. It is
// primarily intended for use by generated code.
func NewZwlrDataControlManagerV1(state wire.State) *ZwlrDataControlManagerV1 {
	return &ZwlrDataControlManagerV1{state: state}
}

func BindZwlrDataControlManagerV1(state wire.State, registry wire.Binder, name, version uint32) *ZwlrDataControlManagerV1 {
	obj := NewZwlrDataControlManagerV1(state)
	state.Add(obj)
	registry.Bind(name, wire.NewID{Interface: ZwlrDataControlManagerV1Interface, Version: version, ID: obj.ID()})
	return obj
}

func (obj *ZwlrDataControlManagerV1) State() wire.State {
	return obj.state
}

func (obj *ZwlrDataControlManagerV1) Dispatch(msg *wire.MessageBuffer) error {

	return wire.UnknownOpError{
		Interface: "zwlr_data_control_manager_v1",
		Type:      "event",
		Op:        msg.Op(),
	}
}

func (obj *ZwlrDataControlManagerV1) ID() uint32 {
	return obj.id
}

func (obj *ZwlrDataControlManagerV1) SetID(id uint32) {
	obj.id = id
}

func (obj *ZwlrDataControlManagerV1) Delete() {
	if obj.OnDelete != nil {
		obj.OnDelete()
	}
}

func (obj *ZwlrDataControlManagerV1) String() string {
	return fmt.Sprintf("%v(%v)", "zwlr_data_control_manager_v1", obj.id)
}

func (obj *ZwlrDataControlManagerV1) MethodName(op uint16) string {
	switch op {
	}

	return "unknown method"
}

func (obj *ZwlrDataControlManagerV1) Interface() string {
	return ZwlrDataControlManagerV1Interface
}

func (obj *ZwlrDataControlManagerV1) Version() uint32 {
	return ZwlrDataControlManagerV1Version
}

// Create a new data source.
func (obj *ZwlrDataControlManagerV1) CreateDataSource() (id *ZwlrDataControlSourceV1) {
	builder := wire.NewMessage(obj, 0)

	id = NewZwlrDataControlSourceV1(obj.state)
	obj.state.Add(id)
	builder.WriteObject(id)

	builder.Method = "create_data_source"
	builder.Args = []any{id}
	obj.state.Enqueue(builder)
	return id
}

// Create a data device that can be used to manage a seat's selection.
func (obj *ZwlrDataControlManagerV1) GetDataDevice(seat *wl.Seat) (id *ZwlrDataControlDeviceV1) {
	builder := wire.NewMessage(obj, 1)

	id = NewZwlrDataControlDeviceV1(obj.state)
	obj.state.Add(id)
	builder.WriteObject(id)
	builder.WriteObject(seat)

	builder.Method = "get_data_device"
	builder.Args = []any{id, seat}
	obj.state.Enqueue(builder)
	return id
}

// All objects created by the manager will still remain valid, until their
// appropriate destroy request has been called.
func (obj *ZwlrDataControlManagerV1) Destroy() {
	builder := wire.NewMessage(obj, 2)

	builder.Method = "destroy"
	builder.Args = []any{}
	obj.state.Enqueue(builder)
	return
}

const (
	ZwlrDataControlDeviceV1Interface = "zwlr_data_control_device_v1"
	ZwlrDataControlDeviceV1Version   = 2
)

// ZwlrDataControlDeviceV1Listener is a type that can respond to incoming
// messages for a ZwlrDataControlDeviceV1 object.
type ZwlrDataControlDeviceV1Listener interface {
	// The data_offer event introduces a new wlr_data_control_offer object,
	// which will subsequently be used in either the
	// wlr_data_control_device.selection event (for the regular clipboard
	// selections) or the wlr_data_control_device.primary_selection event (for
	// the primary clipboard selections). Immediately following the
	// wlr_data_control_device.data_offer event, the new data_offer object
	// will send out wlr_data_control_offer.offer events to describe the MIME
	// types it offers.
	DataOffer(id *ZwlrDataControlOfferV1)

	// The selection event is sent out to notify the client of a new
	// wlr_data_control_offer for the selection for this device. The
	// wlr_data_control_device.data_offer and the wlr_data_control_offer.offer
	// events are sent out immediately before this event to introduce the data
	// offer object. The selection event is sent to a client when a new
	// selection is set. The wlr_data_control_offer is valid until a new
	// wlr_data_control_offer or NULL is received. The client must destroy the
	// previous selection wlr_data_control_offer, if any, upon receiving this
	// event.
	//
	// The first selection event is sent upon binding the
	// wlr_data_control_device object.
	Selection(id *ZwlrDataControlOfferV1)

	// This data control object is no longer valid and should be destroyed by
	// the client.
	Finished()

	// The primary_selection event is sent out to notify the client of a new
	// wlr_data_control_offer for the primary selection for this device. The
	// wlr_data_control_device.data_offer and the wlr_data_control_offer.offer
	// events are sent out immediately before this event to introduce the data
	// offer object. The primary_selection event is sent to a client when a
	// new primary selection is set. The wlr_data_control_offer is valid until
	// a new wlr_data_control_offer or NULL is received. The client must
	// destroy the previous primary selection wlr_data_control_offer, if any,
	// upon receiving this event.
	//
	// If the compositor supports primary selection, the first
	// primary_selection event is sent upon binding the
	// wlr_data_control_device object.
	PrimarySelection(id *ZwlrDataControlOfferV1)
}

// This interface allows a client to manage a seat's selection.
//
// When the seat is destroyed, this object becomes inert.
type ZwlrDataControlDeviceV1 struct {
	// Listener's methods are called by incoming messages from the
	// remote end via Dispatch. If it is nil, messages are silently
	// ignored.
	Listener ZwlrDataControlDeviceV1Listener

	// OnDelete is called when the object is removed from the tracking
	// system.
	OnDelete func()

	state wire.State
	id    uint32
}

// NewZwlrDataControlDeviceV1 returns a newly instantiated ZwlrDataControlDeviceV1. It is
// primarily intended for use by generated code.
func NewZwlrDataControlDeviceV1(state wire.State) *ZwlrDataControlDeviceV1 {
	return &ZwlrDataControlDeviceV1{state: state}
}

func (obj *ZwlrDataControlDeviceV1) State() wire.State {
	return obj.state
}

func (obj *ZwlrDataControlDeviceV1) Dispatch(msg *wire.MessageBuffer) error {
	switch msg.Op() {
	case 0:

		id := NewZwlrDataControlOfferV1(obj.state)
		id.SetID(msg.ReadUint())

		obj.state.Add(id)

		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.DataOffer(
			id,
		)
		return nil

	case 1:

		id, ok := obj.state.Get(msg.ReadUint()).(*ZwlrDataControlOfferV1)
		if !ok {
			return nil
		}

		obj.state.Add(id)

		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.Selection(
			id,
		)
		return nil

	case 2:
		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.Finished()
		return nil

	case 3:
		id, ok := obj.state.Get(msg.ReadUint()).(*ZwlrDataControlOfferV1)
		if !ok {
			return nil
		}

		obj.state.Add(id)

		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.PrimarySelection(
			id,
		)
		return nil
	}

	return wire.UnknownOpError{
		Interface: "zwlr_data_control_device_v1",
		Type:      "event",
		Op:        msg.Op(),
	}
}

func (obj *ZwlrDataControlDeviceV1) ID() uint32 {
	return obj.id
}

func (obj *ZwlrDataControlDeviceV1) SetID(id uint32) {
	obj.id = id
}

func (obj *ZwlrDataControlDeviceV1) Delete() {
	if obj.OnDelete != nil {
		obj.OnDelete()
	}
}

func (obj *ZwlrDataControlDeviceV1) String() string {
	return fmt.Sprintf("%v(%v)", "zwlr_data_control_device_v1", obj.id)
}

func (obj *ZwlrDataControlDeviceV1) MethodName(op uint16) string {
	switch op {
	case 0:
		return "data_offer"

	case 1:
		return "selection"

	case 2:
		return "finished"

	case 3:
		return "primary_selection"
	}

	return "unknown method"
}

func (obj *ZwlrDataControlDeviceV1) Interface() string {
	return ZwlrDataControlDeviceV1Interface
}

func (obj *ZwlrDataControlDeviceV1) Version() uint32 {
	return ZwlrDataControlDeviceV1Version
}

// This request asks the compositor to set the selection to the data from
// the source on behalf of the client.
//
// The given source may not be used in any further set_selection or
// set_primary_selection requests. Attempting to use a previously used
// source is a protocol error.
//
// To unset the selection, set the source to NULL.
func (obj *ZwlrDataControlDeviceV1) SetSelection(source *ZwlrDataControlSourceV1) {
	builder := wire.NewMessage(obj, 0)

	builder.WriteObject(source)

	builder.Method = "set_selection"
	builder.Args = []any{source}
	obj.state.Enqueue(builder)
	return
}

// Destroys the data device object.
func (obj *ZwlrDataControlDeviceV1) Destroy() {
	builder := wire.NewMessage(obj, 1)

	builder.Method = "destroy"
	builder.Args = []any{}
	obj.state.Enqueue(builder)
	return
}

// This request asks the compositor to set the primary selection to the
// data from the source on behalf of the client.
//
// The given source may not be used in any further set_selection or
// set_primary_selection requests. Attempting to use a previously used
// source is a protocol error.
//
// To unset the primary selection, set the source to NULL.
//
// The compositor will ignore this request if it does not support primary
// selection.
func (obj *ZwlrDataControlDeviceV1) SetPrimarySelection(source *ZwlrDataControlSourceV1) {
	builder := wire.NewMessage(obj, 2)

	builder.WriteObject(source)

	builder.Method = "set_primary_selection"
	builder.Args = []any{source}
	obj.state.Enqueue(builder)
	return
}

type ZwlrDataControlDeviceV1Error int64

const (
	// source given to set_selection or set_primary_selection was already used before
	ZwlrDataControlDeviceV1ErrorUsedSource ZwlrDataControlDeviceV1Error = 1
)

func (enum ZwlrDataControlDeviceV1Error) String() string {
	switch enum {
	case 1:
		return "ZwlrDataControlDeviceV1ErrorUsedSource"
	}

	return "<invalid ZwlrDataControlDeviceV1Error>"
}

const (
	ZwlrDataControlSourceV1Interface = "zwlr_data_control_source_v1"
	ZwlrDataControlSourceV1Version   = 1
)

// ZwlrDataControlSourceV1Listener is a type that can respond to incoming
// messages for a ZwlrDataControlSourceV1 object.
type ZwlrDataControlSourceV1Listener interface {
	// Request for data from the client. Send the data as the specified MIME
	// type over the passed file descriptor, then close it.
	Send(mimeType string, fd int)

	// This data source is no longer valid. The data source has been replaced
	// by another data source.
	//
	// The client should clean up and destroy this data source.
	Cancelled()
}

// The wlr_data_control_source object is the source side of a
// wlr_data_control_offer. It is created by the source client in a data
// transfer and provides a way to describe the offered data and a way to
// respond to requests to transfer the data.
type ZwlrDataControlSourceV1 struct {
	// Listener's methods are called by incoming messages from the
	// remote end via Dispatch. If it is nil, messages are silently
	// ignored.
	Listener ZwlrDataControlSourceV1Listener

	// OnDelete is called when the object is removed from the tracking
	// system.
	OnDelete func()

	state wire.State
	id    uint32
}

// NewZwlrDataControlSourceV1 returns a newly instantiated ZwlrDataControlSourceV1. It is
// primarily intended for use by generated code.
func NewZwlrDataControlSourceV1(state wire.State) *ZwlrDataControlSourceV1 {
	return &ZwlrDataControlSourceV1{state: state}
}

func (obj *ZwlrDataControlSourceV1) State() wire.State {
	return obj.state
}

func (obj *ZwlrDataControlSourceV1) Dispatch(msg *wire.MessageBuffer) error {
	switch msg.Op() {
	case 0:

		mimeType := msg.ReadString()

		fd := msg.ReadFile()

		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.Send(
			mimeType,
			int(fd.Fd()),
		)
		return nil

	case 1:
		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.Cancelled()
		return nil
	}

	return wire.UnknownOpError{
		Interface: "zwlr_data_control_source_v1",
		Type:      "event",
		Op:        msg.Op(),
	}
}

func (obj *ZwlrDataControlSourceV1) ID() uint32 {
	return obj.id
}

func (obj *ZwlrDataControlSourceV1) SetID(id uint32) {
	obj.id = id
}

func (obj *ZwlrDataControlSourceV1) Delete() {
	if obj.OnDelete != nil {
		obj.OnDelete()
	}
}

func (obj *ZwlrDataControlSourceV1) String() string {
	return fmt.Sprintf("%v(%v)", "zwlr_data_control_source_v1", obj.id)
}

func (obj *ZwlrDataControlSourceV1) MethodName(op uint16) string {
	switch op {
	case 0:
		return "send"

	case 1:
		return "cancelled"
	}

	return "unknown method"
}

func (obj *ZwlrDataControlSourceV1) Interface() string {
	return ZwlrDataControlSourceV1Interface
}

func (obj *ZwlrDataControlSourceV1) Version() uint32 {
	return ZwlrDataControlSourceV1Version
}

// This request adds a MIME type to the set of MIME types advertised to
// targets. Can be called several times to offer multiple types.
//
// Calling this after wlr_data_control_device.set_selection is a protocol
// error.
func (obj *ZwlrDataControlSourceV1) Offer(mimeType string) {
	builder := wire.NewMessage(obj, 0)

	builder.WriteString(mimeType)

	builder.Method = "offer"
	builder.Args = []any{mimeType}
	obj.state.Enqueue(builder)
	return
}

// Destroys the data source object.
func (obj *ZwlrDataControlSourceV1) Destroy() {
	builder := wire.NewMessage(obj, 1)

	builder.Method = "destroy"
	builder.Args = []any{}
	obj.state.Enqueue(builder)
	return
}

type ZwlrDataControlSourceV1Error int64

const (
	// offer sent after wlr_data_control_device.set_selection
	ZwlrDataControlSourceV1ErrorInvalidOffer ZwlrDataControlSourceV1Error = 1
)

func (enum ZwlrDataControlSourceV1Error) String() string {
	switch enum {
	case 1:
		return "ZwlrDataControlSourceV1ErrorInvalidOffer"
	}

	return "<invalid ZwlrDataControlSourceV1Error>"
}

const (
	ZwlrDataControlOfferV1Interface = "zwlr_data_control_offer_v1"
	ZwlrDataControlOfferV1Version   = 1
)

// ZwlrDataControlOfferV1Listener is a type that can respond to incoming
// messages for a ZwlrDataControlOfferV1 object.
type ZwlrDataControlOfferV1Listener interface {
	// Sent immediately after creating the wlr_data_control_offer object.
	// One event per offered MIME type.
	Offer(mimeType string)
}

// A wlr_data_control_offer represents a piece of data offered for transfer
// by another client (the source client). The offer describes the different
// MIME types that the data can be converted to and provides the mechanism
// for transferring the data directly from the source client.
type ZwlrDataControlOfferV1 struct {
	// Listener's methods are called by incoming messages from the
	// remote end via Dispatch. If it is nil, messages are silently
	// ignored.
	Listener ZwlrDataControlOfferV1Listener

	// OnDelete is called when the object is removed from the tracking
	// system.
	OnDelete func()

	state wire.State
	id    uint32
}

// NewZwlrDataControlOfferV1 returns a newly instantiated ZwlrDataControlOfferV1. It is
// primarily intended for use by generated code.
func NewZwlrDataControlOfferV1(state wire.State) *ZwlrDataControlOfferV1 {
	return &ZwlrDataControlOfferV1{state: state}
}

func (obj *ZwlrDataControlOfferV1) State() wire.State {
	return obj.state
}

func (obj *ZwlrDataControlOfferV1) Dispatch(msg *wire.MessageBuffer) error {
	switch msg.Op() {
	case 0:

		mimeType := msg.ReadString()

		if err := msg.Err(); err != nil {
			return err
		}

		if obj.Listener == nil {
			return nil
		}
		obj.Listener.Offer(
			mimeType,
		)
		return nil
	}

	return wire.UnknownOpError{
		Interface: "zwlr_data_control_offer_v1",
		Type:      "event",
		Op:        msg.Op(),
	}
}

func (obj *ZwlrDataControlOfferV1) ID() uint32 {
	return obj.id
}

func (obj *ZwlrDataControlOfferV1) SetID(id uint32) {
	obj.id = id
}

func (obj *ZwlrDataControlOfferV1) Delete() {
	if obj.OnDelete != nil {
		obj.OnDelete()
	}
}

func (obj *ZwlrDataControlOfferV1) String() string {
	return fmt.Sprintf("%v(%v)", "zwlr_data_control_offer_v1", obj.id)
}

func (obj *ZwlrDataControlOfferV1) MethodName(op uint16) string {
	switch op {
	case 0:
		return "offer"
	}

	return "unknown method"
}

func (obj *ZwlrDataControlOfferV1) Interface() string {
	return ZwlrDataControlOfferV1Interface
}

func (obj *ZwlrDataControlOfferV1) Version() uint32 {
	return ZwlrDataControlOfferV1Version
}

// To transfer the offered data, the client issues this request and
// indicates the MIME type it wants to receive. The transfer happens
// through the passed file descriptor (typically created with the pipe
// system call). The source client writes the data in the MIME type
// representation requested and then closes the file descriptor.
//
// The receiving client reads from the read end of the pipe until EOF and
// then closes its end, at which point the transfer is complete.
//
// This request may happen multiple times for different MIME types.
func (obj *ZwlrDataControlOfferV1) Receive(mimeType string, fd *os.File) {
	builder := wire.NewMessage(obj, 0)

	builder.WriteString(mimeType)
	builder.WriteFile(fd)

	builder.Method = "receive"
	builder.Args = []any{mimeType, fd}
	obj.state.Enqueue(builder)
	return
}

// Destroys the data offer object.
func (obj *ZwlrDataControlOfferV1) Destroy() {
	builder := wire.NewMessage(obj, 1)

	builder.Method = "destroy"
	builder.Args = []any{}
	obj.state.Enqueue(builder)
	return
}
